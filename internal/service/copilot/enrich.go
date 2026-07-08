// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package service

import (
	"context"
	"encoding/base64"
	"io"
	"strings"

	"github.com/lin-snow/ech0/internal/agent"
	echoModel "github.com/lin-snow/ech0/internal/model/echo"
	embeddingModel "github.com/lin-snow/ech0/internal/model/embedding"
	fileModel "github.com/lin-snow/ech0/internal/model/file"
	"github.com/lin-snow/ech0/internal/storage"
)

// maxChatImages 是单轮注入模型的图片数上限（控制 payload 与成本）。
const maxChatImages = 4

// maxImageBytes 是单张图注入的字节上限；超过则跳过（避免超大图撑爆请求）。
const maxImageBytes = 5 << 20 // 5MB

// enrichHits 按命中顺序（最相关在前）回查 Echo（GetEchoById 带缓存），一次加载取齐三样：
//   - exts：每条命中的 Extension 渲染文本（音乐/网站/位置等分享，常开，喂给模型理解）；
//   - results[i].Files：命中 Echo 的媒体附件元数据（图片/视频/音频，常开，随 SSE 给前端展示，仅元数据不含字节）；
//   - images：配图的 base64 ImagePart，仅 multimodal 开启时收集，累计到 maxChatImages 即止（喂模型）。
//
// 不存进 embedding 索引、只在检索命中后回查，向量库保持纯文本干净。读取失败静默跳过（best-effort）。
func (s *CopilotService) enrichHits(
	ctx context.Context,
	results []embeddingModel.SearchResult,
	multimodal bool,
) (map[string]string, []agent.ImagePart) {
	exts := make(map[string]string, len(results))
	var images []agent.ImagePart
	for i := range results {
		echo, err := s.echoService.GetEchoById(ctx, results[i].EchoID)
		if err != nil || echo == nil {
			continue
		}
		if txt := formatExtension(echo.Extension); txt != "" {
			exts[results[i].EchoID] = txt
		}
		results[i].Extension = echo.Extension // 前端展示用：扩展类型标签（音乐/网站/位置…）

		var files []fileModel.File
		for _, ef := range echo.EchoFiles {
			cat := storage.NormalizeCategory(ef.File.Category)
			// 展示用：图片/视频/音频都带给前端（sources 里按类型展示缩略图或类型标志）。
			switch cat {
			case storage.CategoryImage, storage.CategoryVideo, storage.CategoryAudio:
				files = append(files, ef.File) // 整条 File（含 storage_type/key/url/category 等）
			}
			// 多模态：仅图片读成 base64 喂给模型（受 maxChatImages 上限约束）；视频/音频不入模型。
			if cat.IsImageLike() && multimodal && s.storage != nil && len(images) < maxChatImages {
				if part, ok := s.loadImagePart(ctx, ef.File); ok {
					images = append(images, part)
				}
			}
		}
		results[i].Files = files
	}
	return exts, images
}

// formatExtension 把 Echo 的扩展分享渲染成一行供模型理解的文本（无扩展或缺字段返回空）。
// Payload 由 GORM json serializer 反序列化为 map，字符串值原样取出。
func formatExtension(ext *echoModel.EchoExtension) string {
	if ext == nil {
		return ""
	}
	str := func(k string) string {
		if v, ok := ext.Payload[k].(string); ok {
			return strings.TrimSpace(v)
		}
		return ""
	}
	switch ext.Type {
	case echoModel.Extension_MUSIC:
		if u := str("url"); u != "" {
			return "[音乐分享] " + u
		}
	case echoModel.Extension_VIDEO:
		if id := str("videoId"); id != "" {
			return "[视频分享] 视频ID " + id
		}
	case echoModel.Extension_GITHUBPROJ:
		if u := str("repoUrl"); u != "" {
			return "[GitHub 项目] " + u
		}
	case echoModel.Extension_WEBSITE:
		title, site := str("title"), str("site")
		switch {
		case title != "" && site != "":
			return "[网站] " + title + " " + site
		case site != "":
			return "[网站] " + site
		case title != "":
			return "[网站] " + title
		}
	case echoModel.Extension_LOCATION:
		if place := str("placeholder"); place != "" {
			return "[位置] " + place
		}
	case echoModel.Extension_TWEET:
		u, user := str("url"), str("username")
		switch {
		case u != "" && user != "":
			return "[X 推文] @" + user + " " + u
		case u != "":
			return "[X 推文] " + u
		}
	}
	return ""
}

// loadImagePart 把单个 File 读成 ImagePart：external 直接用公网直链；local/object 读字节做 base64。
// 非图片、超限、读失败均返回 ok=false 由调用方跳过。
func (s *CopilotService) loadImagePart(ctx context.Context, f fileModel.File) (agent.ImagePart, bool) {
	if !storage.NormalizeCategory(f.Category).IsImageLike() {
		return agent.ImagePart{}, false
	}
	mediaType := f.ContentType
	if mediaType == "" {
		mediaType = "image/jpeg"
	}

	st := storage.NormalizeStorageType(f.StorageType)
	if st == storage.StorageTypeExternal {
		if f.URL == "" {
			return agent.ImagePart{}, false
		}
		return agent.ImagePart{MediaType: mediaType, URL: f.URL}, true
	}

	if f.Size > maxImageBytes {
		return agent.ImagePart{}, false
	}
	reader, err := s.storage.GetSelector().Get(ctx, st, f.Key)
	if err != nil {
		return agent.ImagePart{}, false
	}
	defer func() { _ = reader.Close() }()
	data, err := io.ReadAll(io.LimitReader(reader, maxImageBytes))
	if err != nil || len(data) == 0 {
		return agent.ImagePart{}, false
	}
	return agent.ImagePart{MediaType: mediaType, Base64: base64.StdEncoding.EncodeToString(data)}, true
}
