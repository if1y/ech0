// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package model

// Heatmap 用于存储热力图数据
type Heatmap struct {
	Date  string `json:"date"`  // 日期
	Count int    `json:"count"` // Echo数量
}

type (
	UploadFileType string
	S3Provider     string
	OAuth2Provider string
	AgentProtocol  string
	Locale         string
)

const (
	// ImageType  图片类型
	ImageType UploadFileType = "image"
	// AudioType  音频类型
	AudioType UploadFileType = "audio"
	// VideoType  视频类型
	VideoType UploadFileType = "video"
)

const (
	AWS     S3Provider = "aws"
	ALIYUN  S3Provider = "aliyun"
	TENCENT S3Provider = "tencent"
	R2      S3Provider = "r2"
	MINIO   S3Provider = "minio"
	OTHER   S3Provider = "other"
)

const (
	OAuth2GITHUB OAuth2Provider = "github"
	OAuth2GOOGLE OAuth2Provider = "google"
	OAuth2QQ     OAuth2Provider = "qq"
	OAuth2CUSTOM OAuth2Provider = "custom"
)

const (
	// OpenAI 协议（覆盖 OpenAI 官方、DeepSeek、Qwen、Ollama 以及任意 OpenAI 兼容服务）
	OpenAI AgentProtocol = "openai"
	// Anthropic 协议
	Anthropic AgentProtocol = "anthropic"
)

const (
	LocaleZhCN     Locale = "zh-CN"
	LocaleEnUS     Locale = "en-US"
	LocaleDeDE     Locale = "de-DE"
	DefaultLocale         = LocaleZhCN
	FallbackLocale        = LocaleEnUS
)

// key value表
type KeyValue struct {
	Key   string `json:"key"   gorm:"primaryKey"`
	Value string `json:"value"`
}

// 键值对相关
const (
	// SystemSettingsKey 是系统设置的键
	SystemSettingsKey = "system_settings"
	// CommentSettingKey 是评论设置的建
	CommentSettingKey = "comment_setting"
	// S3SettingKey 是 S3 存储设置的键
	S3SettingKey = "s3_setting"
	// OAuth2SettingKey 是 OAuth2 设置的键
	OAuth2SettingKey = "oauth2_setting"
	// PasskeySettingKey 是 Passkey 设置的键
	PasskeySettingKey = "passkey_setting"
	// ServerURLKey 是服务器URL设置的键
	ServerURLKey = "server_url"
	// SnapshotScheduleKey 是定时快照计划设置的键
	SnapshotScheduleKey = "snapshot_schedule"
	// AgentSettingKey 是 Agent 设置的键
	AgentSettingKey = "agent_setting"
	// EmbeddingSettingKey 是 Embedding 向量设置的键
	EmbeddingSettingKey = "embedding_setting"
	// EmbeddingIndexStateKey 记录当前已建索引所用的 model/dim（换模型后据此判定需重建）
	EmbeddingIndexStateKey = "embedding_index_state"
	// ReleaseVersionKey 是发布版本号的键
	ReleaseVersionKey = "release_version"
	// InstallInitializedKey 是安装流程完成状态键
	InstallInitializedKey = "install_initialized"
	// MigrationGlobalJobStateKey 是全局迁移作业状态键
	MigrationGlobalJobStateKey = "migration_global_job_state"
	// StorageTimeUTCNormalizedKey 是存储层时间统一为 UTC 的幂等标记键
	StorageTimeUTCNormalizedKey = "storage_time_utc_normalized_v1"
	// StorageTimeSanitizedKey 是存储层时间脏数据安全修复阶段的幂等标记键
	StorageTimeSanitizedKey = "storage_time_sanitized_v1"
	// StorageTimeValidatedKey 是存储层时间可转换性校验阶段的幂等标记键
	StorageTimeValidatedKey = "storage_time_validated_v1"
	// StorageTimeUnixMigratedKey 是存储层时间迁移为 Unix 秒级时间戳的幂等标记键
	StorageTimeUnixMigratedKey = "storage_time_unix_migrated_v1"
	// StorageTimeSchemaRebuiltKey 是存储层时间字段 schema 对齐阶段的幂等标记键
	StorageTimeSchemaRebuiltKey = "storage_time_schema_rebuilt_v1"
	// OAuthBindingsDroppedKey 是历史 oauth_bindings 表清理迁移的幂等标记键
	OAuthBindingsDroppedKey = "oauth_bindings_dropped_v1"
	// LegacyInboxesDroppedKey 是已废弃的 inboxes 表清理迁移的幂等标记键
	LegacyInboxesDroppedKey = "legacy_inboxes_dropped_v1"
	// AgentProtocolCollapsedKey 是 Agent 协议枚举从 7 项收敛为 3 项的幂等标记键
	AgentProtocolCollapsedKey = "agent_provider_collapsed_v1"
	// AgentSettingProtocolRenamedKey 是把 agent_setting JSON 里 "provider" 字段重命名为 "protocol" 的幂等标记键
	AgentSettingProtocolRenamedKey = "agent_setting_protocol_renamed_v1"
	// UserLocalAuthBackfilledKey 是把存量 users.password 回填进 user_local_auth 表的幂等标记键
	UserLocalAuthBackfilledKey = "user_local_auth_backfilled_v1"
	// UsersPasswordColumnDroppedKey 是回填后删除 users.password 遗留列的幂等标记键
	UsersPasswordColumnDroppedKey = "users_password_column_dropped_v1"
	// ChatSessionKeyPrefix 是 Chat 持久化会话的键前缀（每个 userID 一条，键为前缀 + userID）
	ChatSessionKeyPrefix = "chat_session:"
)

// PageQueryResult 用于分页查询的结果数据传输对象
type PageQueryResult[T any] struct {
	Total int64 `json:"total"`
	Items T     `json:"items"`
}
