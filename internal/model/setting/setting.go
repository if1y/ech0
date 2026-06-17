// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package model

import (
	uuidUtil "github.com/lin-snow/ech0/internal/util/uuid"
	"gorm.io/gorm"
)

const (
	EIGHT_HOUR_EXPIRY string = "8_hours"
	ONE_MONTH_EXPIRY  string = "1_month"
	NEVER_EXPIRY      string = "never"
)

// SystemSetting 定义系统设置实体
type SystemSetting struct {
	SiteTitle     string `json:"site_title"`     // 站点标题
	ServerLogo    string `json:"server_logo"`    // 服务器Logo
	ServerName    string `json:"server_name"`    // 服务器名称
	ServerURL     string `json:"server_url"`     // 服务器地址
	AllowRegister bool   `json:"allow_register"` // 是否允许注册'
	DefaultLocale string `json:"default_locale"` // 站点默认语言（如 zh-CN / en-US）
	ICPNumber     string `json:"ICP_number"`     // 备案号
	FooterContent string `json:"footer_content"` // 自定义页脚内容
	FooterLink    string `json:"footer_link"`    // 自定义页脚链接
	MetingAPI     string `json:"meting_api"`     // Meting API 地址
	CustomCSS     string `json:"custom_css"`     // 自定义 CSS
	CustomJS      string `json:"custom_js"`      // 自定义 JS
}

// S3Setting 定义 S3 存储设置实体
type S3Setting struct {
	Enable     bool   `json:"enable"`      // 是否启用 S3 存储
	Provider   string `json:"provider"`    // S3 服务提供商，例如 "aws", "r2", "minio", "other"
	Endpoint   string `json:"endpoint"`    // S3 端点
	AccessKey  string `json:"access_key"`  // 访问密钥 ID
	SecretKey  string `json:"secret_key"`  // 秘密访问密钥
	BucketName string `json:"bucket_name"` // 存储桶名称
	Region     string `json:"region"`      // 区域
	UseSSL     bool   `json:"use_ssl"`     // 是否使用 SSL
	CDNURL     string `json:"cdn_url"`     // CDN 加速域名（可选，没有就走 Endpoint）
	PathPrefix string `json:"path_prefix"` // 存储路径前缀，例如 "uploads/"，方便隔离目录
	NameFormat string `json:"name_format"` // 文件命名规则，支持变量 {year} {month} {day} {hour} {minute} {second} {millisecond} {filename} {timestamp} {userid} {userid:8} {hex:8}
	PublicRead bool   `json:"public_read"` // 上传时是否默认设置对象为 public-read
}

// OAuth2Setting 定义 OAuth2 配置结构体
type OAuth2Setting struct {
	Enable       bool     `json:"enable"`        // 是否启用 OAuth2 登录
	Provider     string   `json:"provider"`      // OAuth2 提供商
	ClientID     string   `json:"client_id"`     // OAuth2 Client ID
	ClientSecret string   `json:"client_secret"` // OAuth2 Client Secret
	RedirectURI  string   `json:"redirect_uri"`  // OAuth2 重定向 URI
	Scopes       []string `json:"scopes"`        // OAuth2 请求的权限范围
	AuthURL      string   `json:"auth_url"`      // OAuth2 授权 URL
	TokenURL     string   `json:"token_url"`     // OAuth2 令牌 URL
	UserInfoURL  string   `json:"user_info_url"` // OAuth2 用户信息 URL

	// OIDC 扩展
	IsOIDC  bool   `json:"is_oidc"`  // 是否启用 OIDC
	Issuer  string `json:"issuer"`   // OIDC 颁发者
	JWKSURL string `json:"jwks_url"` // OIDC JWKS URL

	// 认证边界配置（Panel 主配置，ENV 仅默认值）
	AuthRedirectAllowedReturnURLs []string `json:"auth_redirect_allowed_return_urls"`
	CORSAllowedOrigins            []string `json:"cors_allowed_origins"`
}

// PasskeySetting 定义 Passkey(WebAuthn) 配置结构体
type PasskeySetting struct {
	WebAuthnRPID           string   `json:"webauthn_rp_id"`
	WebAuthnAllowedOrigins []string `json:"webauthn_allowed_origins"`
}

// AccessTokenSetting 定义访问令牌设置实体
type AccessTokenSetting struct {
	ID         string `gorm:"type:char(36);primaryKey" json:"id"`         // 访问令牌 ID
	UserID     string `gorm:"type:char(36);index" json:"user_id"`         // 创建该访问令牌的用户 ID
	Token      string `gorm:"type:varchar(255);uniqueIndex" json:"token"` // 访问令牌
	Name       string `json:"name"`                                       // 访问令牌名称
	TokenType  string `gorm:"size:32;index" json:"token_type"`            // 访问令牌类型（access）
	Scopes     string `gorm:"type:text" json:"scopes"`                    // scopes 的 JSON 字符串
	Audience   string `gorm:"size:64;index" json:"audience"`              // token audience
	JTI        string `gorm:"size:64;uniqueIndex" json:"jti"`             // JWT ID
	Expiry     *int64 `json:"expiry"`                                     // 指针类型，NULL 表示永不过期
	LastUsedAt *int64 `json:"last_used_at,omitempty"`                     // 最后一次使用时间
	CreatedAt  int64  `gorm:"autoCreateTime" json:"created_at"`           // 访问令牌创建时间，Unix 秒级时间戳
}

func (a *AccessTokenSetting) BeforeCreate(_ *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuidUtil.MustNewV7()
	}
	return nil
}

// AgentSetting 定义 LLM Agent 设置实体
type AgentSetting struct {
	Enable     bool   `json:"enable"`     // 是否启用 Agent 功能
	Protocol   string `json:"protocol"`   // LLM 接口协议（OpenAI 兼容/Anthropic，OpenAI 兼容覆盖 DeepSeek、Qwen、Ollama 等）
	Model      string `json:"model"`      // LLM 模型名称
	ApiKey     string `json:"api_key"`    // LLM API Key
	Prompt     string `json:"prompt"`     // Agent 额外使用的提示词
	BaseURL    string `json:"base_url"`   // 自定义 API URL（可选）
	Multimodal bool   `json:"multimodal"` // 多模态支持：Chat 检索命中带图 Echo 时，把配图一并递给模型（需所配模型支持视觉）
	// ContextWindow 是模型上下文窗口的 token 数（0=未配置，按保守默认处理）。
	// 用于区间聚合（年终/月度总结）时的取数预算：窗口越大越倾向「一次塞满全部 Echo」，
	// 越小越早转入按月 map-reduce 分层总结。前端以 256k/1m 形式填写、解析成 token 数后存此。
	ContextWindow int `json:"context_window"`
}

type SnapshotSchedule struct {
	Enable         bool   `json:"enable"`          // 是否启用定时快照
	CronExpression string `json:"cron_expression"` // 定时快照的 Cron 表达式
}
