// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

// Package model 定义 Embedding 领域的数据模型。
package model

import (
	echoModel "github.com/lin-snow/ech0/internal/model/echo"
	fileModel "github.com/lin-snow/ech0/internal/model/file"
)

// EchoEmbedding 是 Echo 向量索引的元数据 + 内容快照表（普通表，进 AutoMigrate）。
//
// 真正的向量存于 sqlite-vec 的 vec0 虚表 vec_echo（由 repository 按配置维度懒建，
// 不走 GORM AutoMigrate）。这里冗余保存 content / echo_created / username 快照，
// 使检索结果自包含、无需回查 echos 主表，降低 Chat 检索的耦合。
type EchoEmbedding struct {
	EchoID      string `gorm:"type:char(36);primaryKey" json:"echo_id"`
	ContentHash string `gorm:"type:varchar(64);index"   json:"content_hash"`
	Model       string `gorm:"type:varchar(100)"        json:"model"`
	Dim         int    `gorm:"default:0"                json:"dim"`
	Content     string `gorm:"type:text"                json:"content"`
	Username    string `gorm:"type:varchar(100)"        json:"username"`
	EchoCreated int64  `gorm:"index"                    json:"echo_created"`
	CreatedAt   int64  `gorm:"autoCreateTime"           json:"created_at"`
	UpdatedAt   int64  `gorm:"autoUpdateTime"           json:"updated_at"`
}

// TableName 固定元数据表名。
func (EchoEmbedding) TableName() string { return "echo_embeddings" }

// SearchResult 是一次向量检索命中的结果（含内容快照与距离）。
//
// Files / Extension 是命中 Echo 的媒体附件（图片/视频/音频）与扩展分享，仅在检索命中后回查填充
// （见 copilot.enrichHits），用于前端在引用来源里展示缩略图与类型标志——只随 SSE/会话给前端，
// 不进向量索引、也不喂模型。
type SearchResult struct {
	EchoID      string                   `json:"echo_id"`
	Content     string                   `json:"content"`
	Username    string                   `json:"username"`
	EchoCreated int64                    `json:"echo_created"`
	Distance    float64                  `json:"distance"`
	Files       []fileModel.File         `json:"files,omitempty"`
	Extension   *echoModel.EchoExtension `json:"extension,omitempty"`
}

// IndexState 记录当前已建索引所用的模型与维度（存于 KeyValue，用于换模型检测）。
type IndexState struct {
	Model string `json:"model"`
	Dim   int    `json:"dim"`
}
