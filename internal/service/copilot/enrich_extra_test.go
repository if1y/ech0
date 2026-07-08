// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package service

import (
	"context"
	"reflect"
	"testing"

	echoModel "github.com/lin-snow/ech0/internal/model/echo"
	embeddingModel "github.com/lin-snow/ech0/internal/model/embedding"
	fileModel "github.com/lin-snow/ech0/internal/model/file"
	settingModel "github.com/lin-snow/ech0/internal/model/setting"
	"github.com/lin-snow/ech0/internal/storage"
)

// enrichEchoSvc 是 EchoService 的测试替身，只覆写 GetEchoById 返回带指定附件的 Echo，
// 其余方法未实现（嵌入 nil 接口，未调用即不 panic）。enrichHits 只用到 GetEchoById。
type enrichEchoSvc struct {
	EchoService
	echo *echoModel.Echo
}

func (f *enrichEchoSvc) GetEchoById(_ context.Context, _ string) (*echoModel.Echo, error) {
	return f.echo, nil
}

// enrichHits：命中回查时图片/视频/音频都进 results.Files（供前端展示），pdf/file 等非媒体排除；
// 多模态关闭时不产出 base64 图片。锁住「sources 带媒体类型标志、但不把非媒体塞给前端」的契约。
func TestEnrichHits_MediaCategoriesToFiles(t *testing.T) {
	echoFile := func(cat string) fileModel.EchoFile {
		return fileModel.EchoFile{File: fileModel.File{Category: cat, URL: "https://f/" + cat}}
	}
	svc := &enrichEchoSvc{echo: &echoModel.Echo{
		ID: "e1",
		EchoFiles: []fileModel.EchoFile{
			echoFile(string(storage.CategoryImage)),
			echoFile(string(storage.CategoryVideo)),
			echoFile(string(storage.CategoryAudio)),
			echoFile(string(storage.CategoryPDF)),
			echoFile(string(storage.CategoryFile)),
		},
	}}
	s := &CopilotService{echoService: svc}

	results := []embeddingModel.SearchResult{{EchoID: "e1"}}
	_, images := s.enrichHits(context.Background(), results, false)

	gotCats := make([]string, 0, len(results[0].Files))
	for _, f := range results[0].Files {
		gotCats = append(gotCats, f.Category)
	}
	want := []string{"image", "video", "audio"}
	if !reflect.DeepEqual(gotCats, want) {
		t.Fatalf("Files 类别 = %v, want %v（应带图片/视频/音频，排除 pdf/file）", gotCats, want)
	}
	if len(images) != 0 {
		t.Fatalf("多模态关闭时不应产出 base64 图片，got %d", len(images))
	}
}

// formatExtension：覆盖各扩展类型的渲染分支与缺字段/空值降级。
func TestFormatExtension(t *testing.T) {
	ext := func(typ string, kv map[string]any) *echoModel.EchoExtension {
		return &echoModel.EchoExtension{Type: typ, Payload: kv}
	}
	cases := []struct {
		name string
		in   *echoModel.EchoExtension
		want string
	}{
		{"nil", nil, ""},
		{"music", ext(echoModel.Extension_MUSIC, map[string]any{"url": "https://song"}), "[音乐分享] https://song"},
		{"music missing url", ext(echoModel.Extension_MUSIC, map[string]any{}), ""},
		{"video", ext(echoModel.Extension_VIDEO, map[string]any{"videoId": "abc"}), "[视频分享] 视频ID abc"},
		{"github", ext(echoModel.Extension_GITHUBPROJ, map[string]any{"repoUrl": "https://gh"}), "[GitHub 项目] https://gh"},
		{"website title+site", ext(echoModel.Extension_WEBSITE, map[string]any{"title": "T", "site": "S"}), "[网站] T S"},
		{"website site only", ext(echoModel.Extension_WEBSITE, map[string]any{"site": "S"}), "[网站] S"},
		{"website title only", ext(echoModel.Extension_WEBSITE, map[string]any{"title": "T"}), "[网站] T"},
		{"website empty", ext(echoModel.Extension_WEBSITE, map[string]any{}), ""},
		{"location", ext(echoModel.Extension_LOCATION, map[string]any{"placeholder": "北京"}), "[位置] 北京"},
		{"tweet url+user", ext(echoModel.Extension_TWEET, map[string]any{"url": "https://x", "username": "bob"}), "[X 推文] @bob https://x"},
		{"tweet url only", ext(echoModel.Extension_TWEET, map[string]any{"url": "https://x"}), "[X 推文] https://x"},
		{"unknown type", ext("WHATEVER", map[string]any{"url": "x"}), ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatExtension(c.in); got != c.want {
				t.Fatalf("formatExtension = %q, want %q", got, c.want)
			}
		})
	}
}

// loadImagePart：external 直链命中、缺 URL 跳过、非图片跳过——这些分支不触碰底层存储。
func TestLoadImagePart_NonStorageBranches(t *testing.T) {
	s := &CopilotService{} // storage 为 nil：以下分支均不访问它

	t.Run("external image uses URL", func(t *testing.T) {
		part, ok := s.loadImagePart(context.Background(), fileModel.File{
			Category:    string(storage.CategoryImage),
			StorageType: string(storage.StorageTypeExternal),
			URL:         "https://img/x.png",
			ContentType: "image/png",
		})
		if !ok {
			t.Fatalf("external image should produce an ImagePart")
		}
		if part.URL != "https://img/x.png" || part.MediaType != "image/png" {
			t.Fatalf("unexpected image part: %+v", part)
		}
	})

	t.Run("external without URL is skipped", func(t *testing.T) {
		if _, ok := s.loadImagePart(context.Background(), fileModel.File{
			Category:    string(storage.CategoryImage),
			StorageType: string(storage.StorageTypeExternal),
		}); ok {
			t.Fatalf("external image without URL should be skipped")
		}
	})

	t.Run("non-image is skipped", func(t *testing.T) {
		if _, ok := s.loadImagePart(context.Background(), fileModel.File{
			Category: "AUDIO",
		}); ok {
			t.Fatalf("non-image file should be skipped")
		}
	})
}

// aggregateBudgetTokens / chatContextBudgetTokens：窗口推算与下限保护。
func TestAggregateBudgetTokens(t *testing.T) {
	// 未配置窗口 → 走默认窗口（256k）→ 远高于下限。
	def := aggregateBudgetTokens(settingModel.AgentSetting{ContextWindow: 0})
	if def <= minAggregateBudget {
		t.Fatalf("default-window budget should exceed the floor, got %d", def)
	}
	// 极小窗口 → clamp 到下限。
	if got := aggregateBudgetTokens(settingModel.AgentSetting{ContextWindow: 1000}); got != minAggregateBudget {
		t.Fatalf("tiny window should clamp to floor %d, got %d", minAggregateBudget, got)
	}
	// chatContextBudgetTokens 与聚合预算同口径。
	if chatContextBudgetTokens(settingModel.AgentSetting{ContextWindow: 1000}) != minAggregateBudget {
		t.Fatalf("chat budget should mirror aggregate budget")
	}
}

// runStringsFor：中英 locale 各自取到对应文案集（非空且区分语言）。
func TestRunStringsFor(t *testing.T) {
	zh := runStringsFor("zh-CN")
	if zh.ToolError == "" || zh.UnknownTool == "" {
		t.Fatalf("zh run strings should be populated: %+v", zh)
	}
	en := runStringsFor("en-US")
	if en.UnknownTool == zh.UnknownTool {
		t.Fatalf("en/zh run strings should differ: %q vs %q", en.UnknownTool, zh.UnknownTool)
	}
}
