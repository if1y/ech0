// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package model

// PageQueryDto 用于分页查询的请求数据传输对象
//
// swagger:model PageQueryDto
type PageQueryDto struct {
	Page     int    `json:"page"     form:"page"`
	PageSize int    `json:"pageSize" form:"pageSize"`
	Search   string `json:"search"   form:"search"`
}

// EchoQueryDto 统一 Echo 查询接口的请求体
//
// swagger:model EchoQueryDto
type EchoQueryDto struct {
	Page      int      `json:"page"`
	PageSize  int      `json:"pageSize"`
	Search    string   `json:"search"`
	TagIDs    []string `json:"tagIds"`
	SortBy    string   `json:"sortBy"`
	SortOrder string   `json:"sortOrder"`
	// DateFrom / DateTo：按 echos.created_at 过滤的 Unix 秒闭区间。
	// 0 或负数视为未设置。
	DateFrom int64 `json:"dateFrom"`
	DateTo   int64 `json:"dateTo"`
	// Private：按可见性过滤的三态开关。nil 表示不过滤（现状行为：admin 公开+私密
	// 混合，匿名仅公开）；true 仅私密、false 仅公开。仅当 viewer 具备私密可见权限
	// （admin）时生效，无权限的请求在仓储层被静默忽略、仍强制仅公开。
	Private *bool `json:"private,omitempty"`
	// UserID：按作者（echos.user_id）精确过滤。opt-in——空串表示不限定作者
	// （公开 /echo/query 等调用方留空即保持原行为）；Copilot Chat 用它把检索
	// 收口到当前对话用户本人发布的 Echo。不暴露给前端 JSON 契约，仅服务内部设置。
	UserID string `json:"-"`
}

// FileDto is the unified response for file operations.
// The Key field is the single source of truth — URLs are resolved at runtime.
//
// swagger:model FileDto
type FileDto struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Key         string `json:"key"`
	StorageType string `json:"storage_type,omitempty"`
	URL         string `json:"url"`
	ContentType string `json:"content_type,omitempty"`
	Category    string `json:"category,omitempty"`
	Size        int64  `json:"size,omitempty"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
}

// FileDeleteDto is the request body for deleting a file.
//
// swagger:model FileDeleteDto
type FileDeleteDto struct {
	ID string `json:"id" binding:"required"`
}

// PresignDto 用于响应 S3 预签名 URL 的请求数据传输对象
//
// swagger:model PresignDto
type PresignDto struct {
	ID          string `json:"id"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	Key         string `json:"key"`
	PresignURL  string `json:"presign_url"`
	FileURL     string `json:"file_url"`
}

// GetPresignURLDto 用于请求 S3 预签名 URL 的请求数据传输对象
//
// swagger:model GetPresignURLDto
type GetPresignURLDto struct {
	FileName    string `json:"file_name" binding:"required"`
	ContentType string `json:"content_type"`
	StorageType string `json:"storage_type,omitempty"`
}

// CreateExternalFileDto 用于直链文件入库请求
//
// swagger:model CreateExternalFileDto
type CreateExternalFileDto struct {
	URL         string `json:"url" binding:"required"`
	ContentType string `json:"content_type"`
	Category    string `json:"category"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Name        string `json:"name"`
}

// UpdateFileMetaDto 用于回填对象存储上传后的元信息
//
// swagger:model UpdateFileMetaDto
type UpdateFileMetaDto struct {
	Size        int64  `json:"size" binding:"required,min=0"`
	Width       *int   `json:"width,omitempty"`
	Height      *int   `json:"height,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}

// FileListQueryDto 文件列表查询参数
//
// swagger:model FileListQueryDto
type FileListQueryDto struct {
	Page        int    `json:"page" form:"page"`
	PageSize    int    `json:"pageSize" form:"pageSize"`
	Search      string `json:"search" form:"search"`
	StorageType string `json:"storage_type" form:"storage_type"`
}

// FileListItemDto 文件列表项
//
// swagger:model FileListItemDto
type FileListItemDto struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Key         string `json:"key"`
	StorageType string `json:"storage_type"`
	URL         string `json:"url"`
	ContentType string `json:"content_type,omitempty"`
	Size        int64  `json:"size,omitempty"`
	CreatedAt   int64  `json:"created_at"`
}

// FileListResultDto 文件列表结果
//
// swagger:model FileListResultDto
type FileListResultDto struct {
	Total int64             `json:"total"`
	Items []FileListItemDto `json:"items"`
}

// FileTreeQueryDto 文件树查询参数（懒加载）
//
// swagger:model FileTreeQueryDto
type FileTreeQueryDto struct {
	StorageType string `json:"storage_type" form:"storage_type" binding:"required"`
	Prefix      string `json:"prefix" form:"prefix"`
}

// FilePathStreamQueryDto 按存储路径直接流式读取文件
//
// swagger:model FilePathStreamQueryDto
type FilePathStreamQueryDto struct {
	StorageType string `json:"storage_type" form:"storage_type" binding:"required"`
	Path        string `json:"path" form:"path" binding:"required"`
	Name        string `json:"name" form:"name"`
	ContentType string `json:"content_type" form:"content_type"`
}

// FileTreeNodeDto 文件树节点
//
// swagger:model FileTreeNodeDto
type FileTreeNodeDto struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	NodeType    string `json:"node_type"` // file|folder
	HasChildren bool   `json:"has_children"`
	FileID      string `json:"file_id,omitempty"`
	Size        int64  `json:"size,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	ModifiedAt  int64  `json:"modified_at,omitempty"`
}

// FileTreeResultDto 文件树结果
//
// swagger:model FileTreeResultDto
type FileTreeResultDto struct {
	Items []FileTreeNodeDto `json:"items"`
}

// GetWebsiteTitleDto 用于请求网站标题的请求数据传输对象
//
// swagger:model GetWebsiteTitleDto
type GetWebsiteTitleDto struct {
	WebSiteURL string `json:"website_url" form:"website_url" binding:"required"`
}
