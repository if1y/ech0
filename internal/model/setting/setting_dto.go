// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package model

// SystemSettingDto 定义系统设置数据传输对象
type SystemSettingDto struct {
	SiteTitle        string `json:"site_title"`          // 站点标题
	ServerLogo       string `json:"server_logo"`         // 服务器Logo
	ServerLogoFileID string `json:"server_logo_file_id"` // 服务器Logo文件ID（用于确认临时文件）
	ServerName       string `json:"server_name"`         // 服务器名称
	ServerURL        string `json:"server_url"`          // 服务器地址
	LoaderImageURL   string `json:"loader_image_url"`    // 加载页自定义图片, 空则用笑脸
	LoaderBrandText  string `json:"loader_brand_text"`   // 加载页标题, 空则用"Ech0"
	LoaderSlogan     string `json:"loader_slogan"`       // 加载页口号, 空则用"静候灵感落笔"
	AllowRegister    bool   `json:"allow_register"`      // 是否允许注册
	DefaultLocale    string `json:"default_locale"`      // 站点默认语言（如 zh-CN / en-US）
	ICPNumber        string `json:"ICP_number"`          // 备案号
	FooterContent    string `json:"footer_content"`      // 自定义页脚内容
	FooterLink       string `json:"footer_link"`         // 自定义页脚链接
	MetingAPI        string `json:"meting_api"`          // Meting API 地址
	CustomCSS        string `json:"custom_css"`          // 自定义 CSS
	CustomJS         string `json:"custom_js"`           // 自定义 JS
}

type S3SettingDto struct {
	Enable     bool   `json:"enable"`      // 是否启用 S3 存储
	Provider   string `json:"provider"`    // S3 服务提供商，例如 "aws", "aliyun", "minio", "other"
	Endpoint   string `json:"endpoint"`    // S3 端点
	AccessKey  string `json:"access_key"`  // 访问密钥 ID
	SecretKey  string `json:"secret_key"`  // 秘密访问密钥
	BucketName string `json:"bucket_name"` // 存储桶名称
	Region     string `json:"region"`      // 区域
	UseSSL     bool   `json:"use_ssl"`     // 是否使用 SSL
	CDNURL     string `json:"cdn_url"`     // CDN 加速域名（可选，没有就走 Endpoint）
	PathPrefix string `json:"path_prefix"` // 存储路径前缀，例如 "uploads/"，方便隔离目录
	PublicRead bool   `json:"public_read"` // 上传时是否默认设置对象为 public-read
}

type OAuth2SettingDto struct {
	Enable       bool     `json:"enable"`
	Provider     string   `json:"provider"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURI  string   `json:"redirect_uri"`
	Scopes       []string `json:"scopes"`
	AuthURL      string   `json:"auth_url"`
	TokenURL     string   `json:"token_url"`
	UserInfoURL  string   `json:"user_info_url"`

	IsOIDC  bool   `json:"is_oidc"`  // 是否启用 OIDC
	Issuer  string `json:"issuer"`   // OIDC 颁发者
	JWKSURL string `json:"jwks_url"` // OIDC JWKS URL

	AuthRedirectAllowedReturnURLs []string `json:"auth_redirect_allowed_return_urls"`
	CORSAllowedOrigins            []string `json:"cors_allowed_origins"`
}

type OAuth2Status struct {
	Enabled    bool   `json:"enabled"`
	Provider   string `json:"provider"`
	OAuthReady bool   `json:"oauth_ready"`
}

type PasskeySettingDto struct {
	WebAuthnRPID           string   `json:"webauthn_rp_id"`
	WebAuthnAllowedOrigins []string `json:"webauthn_allowed_origins"`
}

type PasskeyStatus struct {
	PasskeyReady bool `json:"passkey_ready"`
}

type WebhookDto struct {
	Name     string `json:"name"`                                 // Webhook 名称
	URL      string `json:"url"`                                  // Webhook URL
	Secret   string `json:"secret,omitempty"`                     // 签名密钥，用于请求验证（HMAC等）
	IsActive bool   `json:"is_active"        gorm:"default:true"` // 启用/禁用状态
}

type AccessTokenSettingDto struct {
	Name     string   `json:"name"`     // 访问令牌名称
	Expiry   string   `json:"expiry"`   // 访问令牌过期策略（8_hours/1_month/never）
	Scopes   []string `json:"scopes"`   // 访问令牌权限范围
	Audience string   `json:"audience"` // 访问令牌受众（public-client/cli/integration）
}

type SnapshotScheduleDto struct {
	Enable         bool   `json:"enable"`          // 是否启用定时快照
	CronExpression string `json:"cron_expression"` // 定时快照的 Cron 表达式
}

type AgentSettingDto struct {
	Enable     bool   `json:"enable"`     // 是否启用 Agent 功能
	Protocol   string `json:"protocol"`   // LLM 接口协议（OpenAI 兼容/Anthropic，OpenAI 兼容覆盖 DeepSeek、Qwen、Ollama 等）
	Model      string `json:"model"`      // LLM 模型名称
	ApiKey     string `json:"api_key"`    // LLM API Key
	Prompt     string `json:"prompt"`     // Agent 额外使用的提示词
	BaseURL    string `json:"base_url"`   // 自定义 API URL（可选）
	Multimodal bool   `json:"multimodal"` // 多模态支持：检索命中带图 Echo 时把配图递给模型（需模型支持视觉）
	// ContextWindow 是模型上下文窗口的 token 数（0=未配置，按保守默认处理）；用于区间聚合的取数预算。
	ContextWindow int `json:"context_window"`
}
