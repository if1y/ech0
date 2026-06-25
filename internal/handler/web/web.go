// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lin-snow/ech0/internal/kvstore"
	commonModel "github.com/lin-snow/ech0/internal/model/common"
	"github.com/lin-snow/ech0/internal/visitor"
	"github.com/lin-snow/ech0/template"
)

var spaBypassPrefixes = []string{
	"/api",
	"/ws",
	"/mcp",
	"/swagger",
}

// loaderConfig 仅取加载页个性化所需的 3 个字段，从 KV 的 system_settings JSON 中解出。
type loaderConfig struct {
	LoaderImageURL  string `json:"loader_image_url"`
	LoaderBrandText string `json:"loader_brand_text"`
	LoaderSlogan    string `json:"loader_slogan"`
}

type WebHandler struct {
	visitorTracker *visitor.Tracker
	durableKV      kvstore.Store
}

// NewWebHandler WebHandler 的构造函数
func NewWebHandler(visitorTracker *visitor.Tracker, durableKV kvstore.Store) *WebHandler {
	return &WebHandler{
		visitorTracker: visitorTracker,
		durableKV:      durableKV,
	}
}

// Templates 返回一个处理前端编译后文件的 gin.HandlerFunc
func (webHandler *WebHandler) Templates() gin.HandlerFunc {
	// 提取 dist 子目录
	subFS, _ := fs.Sub(template.WebFS, "dist")
	fileServer := http.FS(subFS)

	return func(ctx *gin.Context) {
		requestPath := ctx.Request.URL.Path
		if shouldBypassSPAFallback(requestPath) {
			ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		if requestPath == "/" {
			requestPath = "/index.html"
		}

		if strings.Contains(requestPath, "..") {
			ctx.Status(http.StatusForbidden)
			return
		}

		fullPath := path.Clean("." + requestPath)
		if requestPath == "/index.html" {
			webHandler.serveIndexHTML(ctx, fileServer)
			return
		}
		f, err := fileServer.Open(fullPath)
		if err != nil {
			// SPA fallback → 一律返回 index.html（含注入）
			webHandler.serveIndexHTML(ctx, fileServer)
			return
		}
		defer func() { _ = f.Close() }()

		// 获取文件信息
		stat, _ := f.Stat()

		// 适配资源压缩Gzip 算法
		encoding := ctx.GetHeader("Accept-Encoding")
		if strings.Contains(encoding, "gzip") {
			gzPath := fullPath + ".gz"
			gzFile, err := fileServer.Open(gzPath)
			if err == nil {
				defer func() { _ = gzFile.Close() }()
				stat, _ := gzFile.Stat()
				ctx.Header("Content-Encoding", "gzip")
				ctx.Header("Content-Type", getMimeType(fullPath))
				setCacheControlHeader(ctx, requestPath)
				http.ServeContent(ctx.Writer, ctx.Request, gzPath, stat.ModTime(), gzFile)
				return
			}
		}

		ctx.Header("Content-Type", getMimeType(fullPath))
		setCacheControlHeader(ctx, requestPath)
		http.ServeContent(ctx.Writer, ctx.Request, fullPath, stat.ModTime(), f)
	}
}

// serveIndexHTML 读取 index.html 原始内容，注入 loader 配置后写入响应。
func (webHandler *WebHandler) serveIndexHTML(ctx *gin.Context, fileServer http.FileSystem) {
	fallback, err := fileServer.Open("index.html")
	if err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}
	defer func() { _ = fallback.Close() }()

	fallbackStat, _ := fallback.Stat()
	raw, err := io.ReadAll(fallback)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	// 从 KV 读取 loader 配置
	scriptTag := webHandler.buildLoaderScript()

	// 在 </head> 之前插入 <script> 标签（同步，在 Vue 加载前执行）
	injected := bytes.Replace(raw, []byte("</head>"), []byte(scriptTag+"</head>"), 1)

	webHandler.visitorTracker.Record(ctx.Request, ctx.ClientIP())
	ctx.Header("Content-Type", "text/html; charset=utf-8")
	setCacheControlHeader(ctx, "/index.html")
	http.ServeContent(ctx.Writer, ctx.Request, "index.html", fallbackStat.ModTime(), bytes.NewReader(injected))
}

// buildLoaderScript 从 KV 读取 system_settings，返回 <script> 标签内容。
func (webHandler *WebHandler) buildLoaderScript() string {
	var cfg loaderConfig

	if webHandler.durableKV != nil {
		raw, err := webHandler.durableKV.Get(context.Background(), commonModel.SystemSettingsKey)
		if err == nil && raw != "" {
			// 忽略解析错误，解析失败时 cfg 保持零值（即全部为空字符串，前端不做替换）
			_ = json.Unmarshal([]byte(raw), &cfg)
		}
	}

	// 将 3 个字段序列化成 JSON 对象字符串
	jsonBytes, _ := json.Marshal(cfg)
	jsonStr := string(jsonBytes)

	return fmt.Sprintf(`<script>window.__LOADER_CONFIG__=%s;</script>`, jsonStr)
}

func setCacheControlHeader(ctx *gin.Context, requestPath string) {
	if strings.HasPrefix(requestPath, "/assets/") {
		ctx.Header("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	if requestPath == "/index.html" || requestPath == "/" {
		setNoStoreHeaders(ctx)
		return
	}
	ctx.Header("Cache-Control", "public, max-age=3600")
}

func setNoStoreHeaders(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	ctx.Header("Pragma", "no-cache")
	ctx.Header("Expires", "0")
}

func shouldBypassSPAFallback(requestPath string) bool {
	for _, prefix := range spaBypassPrefixes {
		if requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/") {
			return true
		}
	}
	return false
}

// getMimeType 根据文件扩展名返回 MIME 类型，带默认值
func getMimeType(path string) string {
	ext := filepath.Ext(path)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	return mimeType
}
