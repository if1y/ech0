// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package setting

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/lin-snow/ech0/internal/config"
	i18nUtil "github.com/lin-snow/ech0/internal/i18n"
	"github.com/lin-snow/ech0/internal/kvstore"
	commentModel "github.com/lin-snow/ech0/internal/model/comment"
	commonModel "github.com/lin-snow/ech0/internal/model/common"
	settingModel "github.com/lin-snow/ech0/internal/model/setting"
	urlUtil "github.com/lin-snow/ech0/internal/util/url"
)

// 各配置的「词汇」：一处声明 key / 默认值 / 归一化（/ 升级迁移）。读取统一走
// setting.Get(ctx, kv, setting.Xxx)，seeding 统一由 registry 驱动。
var (
	// System 站点系统设置。默认值取自 config（ENV），URL 字段先 TrimURL。
	System = Spec[settingModel.SystemSetting]{
		Key: commonModel.SystemSettingsKey,
		Default: func() settingModel.SystemSetting {
			c := config.Config().Setting
			return settingModel.SystemSetting{
				SiteTitle:     c.SiteTitle,
				ServerLogo:    c.ServerLogo,
				ServerName:    c.Servername,
				ServerURL:     urlUtil.TrimURL(c.Serverurl),
				AllowRegister: c.AllowRegister,
				DefaultLocale: string(commonModel.DefaultLocale),
				ICPNumber:     c.Icpnumber,
				FooterContent: c.FooterContent,
				FooterLink:    urlUtil.TrimURL(c.FooterLink),
				MetingAPI:     urlUtil.TrimURL(c.MetingAPI),
				CustomCSS:     c.CustomCSS,
				CustomJS:      c.CustomJS,
			}
		},
		Normalize: func(s *settingModel.SystemSetting) {
			s.DefaultLocale = i18nUtil.ResolveLocale(s.DefaultLocale)
		},
	}

	// OAuth2 登录设置。认证边界（returnURL/CORS 白名单）以 Panel 为主、ENV 仅默认值。
	OAuth2 = Spec[settingModel.OAuth2Setting]{
		Key: commonModel.OAuth2SettingKey,
		Default: func() settingModel.OAuth2Setting {
			return settingModel.OAuth2Setting{
				Enable:                        false,
				Provider:                      string(commonModel.OAuth2GITHUB),
				AuthURL:                       "https://github.com/login/oauth/authorize",
				TokenURL:                      "https://github.com/login/oauth/access_token",
				UserInfoURL:                   "https://api.github.com/user",
				Scopes:                        []string{"read:user"},
				AuthRedirectAllowedReturnURLs: append([]string{}, config.Config().Auth.Redirect.AllowedReturnURLs...),
				CORSAllowedOrigins:            append([]string{}, config.Config().Web.CORS.AllowedOrigins...),
			}
		},
		Normalize: normalizeOAuth2Boundary,
	}

	// S3 对象存储设置。默认值取自 config，并做与历史读路径一致的 endpoint/CDN/前缀清洗。
	// 读出后的脱敏（非管理员 ******）属请求态逻辑，留在 SettingService，不在此归一化。
	S3 = Spec[settingModel.S3Setting]{
		Key: commonModel.S3SettingKey,
		Default: func() settingModel.S3Setting {
			c := config.Config().Storage
			return settingModel.S3Setting{
				Enable:     c.ObjectEnabled,
				Provider:   strings.TrimSpace(c.Provider),
				Endpoint:   stripScheme(strings.TrimSpace(c.Endpoint)),
				AccessKey:  c.AccessKey,
				SecretKey:  c.SecretKey,
				BucketName: c.BucketName,
				Region:     strings.TrimSpace(c.Region),
				UseSSL:     c.UseSSL,
				CDNURL:     strings.TrimRight(strings.TrimSpace(c.CDNURL), "/"),
				PathPrefix: strings.Trim(strings.TrimSpace(c.PathPrefix), "/"),
				NameFormat: defaultNameFormat(strings.TrimSpace(c.NameFormat)),
				PublicRead: true,
			}
		},
	}

	// Passkey(WebAuthn) 设置。首次 seeding 时优先从旧 oauth2_setting 搬迁 WebAuthn
	// 字段（Migrate），避免历史用户升级后丢配置；否则回退 config 默认。
	Passkey = Spec[settingModel.PasskeySetting]{
		Key: commonModel.PasskeySettingKey,
		Default: func() settingModel.PasskeySetting {
			return settingModel.PasskeySetting{
				WebAuthnRPID:           strings.TrimSpace(config.Config().Auth.WebAuthn.RPID),
				WebAuthnAllowedOrigins: append([]string{}, config.Config().Auth.WebAuthn.Origins...),
			}
		},
		Normalize: normalizePasskeyBoundary,
		Migrate:   migratePasskeyFromLegacy,
	}

	// Agent LLM 生成设置。
	Agent = Spec[settingModel.AgentSetting]{
		Key: commonModel.AgentSettingKey,
		Default: func() settingModel.AgentSetting {
			return settingModel.AgentSetting{
				Enable:   false,
				Protocol: string(commonModel.OpenAI),
			}
		},
	}

	// Snapshot 定时快照计划。
	Snapshot = Spec[settingModel.SnapshotSchedule]{
		Key: commonModel.SnapshotScheduleKey,
		Default: func() settingModel.SnapshotSchedule {
			return settingModel.SnapshotSchedule{
				Enable:         false,
				CronExpression: "0 2 * * 0", // 每周日凌晨 2 点
			}
		},
	}

	// Embedding 向量设置。默认零值（Enable=false），与历史「miss 即视为未启用」一致。
	Embedding = Spec[settingModel.EmbeddingSetting]{
		Key: commonModel.EmbeddingSettingKey,
		Default: func() settingModel.EmbeddingSetting {
			return settingModel.EmbeddingSetting{Enable: false}
		},
	}

	// Comment 评论系统设置（含邮件通知）。SMTPPassword 的脱敏（SMTPPasswordSet 模式）
	// 属输出投影，留在 CommentService 的读出口，不在此归一化。
	Comment = Spec[commentModel.SystemSetting]{
		Key: commentModel.CommentSystemSettingKey,
		Default: func() commentModel.SystemSetting {
			s := commentModel.SystemSetting{
				EnableComment:   true,
				RequireApproval: true,
				CaptchaEnabled:  false,
			}
			normalizeComment(&s)
			return s
		},
		Normalize: normalizeComment,
	}
)

// registry 是 seeder 的驱动表。serverURLSeed 紧随 System，把派生的 server_url
// 便捷键一并落库。
var registry = []seedable{
	System,
	serverURLSeed{},
	OAuth2,
	S3,
	Passkey,
	Agent,
	Snapshot,
	Embedding,
	Comment,
}

// normalizeOAuth2Boundary 在边界白名单为空时回退到 config 默认（方向 config→value）。
func normalizeOAuth2Boundary(s *settingModel.OAuth2Setting) {
	if len(s.AuthRedirectAllowedReturnURLs) == 0 {
		s.AuthRedirectAllowedReturnURLs = append([]string{}, config.Config().Auth.Redirect.AllowedReturnURLs...)
	}
	if len(s.CORSAllowedOrigins) == 0 {
		s.CORSAllowedOrigins = append([]string{}, config.Config().Web.CORS.AllowedOrigins...)
	}
}

// normalizePasskeyBoundary 在 RPID/Origins 为空时回退到 config 默认。
func normalizePasskeyBoundary(s *settingModel.PasskeySetting) {
	if strings.TrimSpace(s.WebAuthnRPID) == "" {
		s.WebAuthnRPID = strings.TrimSpace(config.Config().Auth.WebAuthn.RPID)
	}
	if len(s.WebAuthnAllowedOrigins) == 0 {
		s.WebAuthnAllowedOrigins = append([]string{}, config.Config().Auth.WebAuthn.Origins...)
	}
}

// normalizeComment 补齐邮件端口默认（与 CommentService.applySettingDefaults 同规则，
// 跨 service/setting 边界不便共享，保留这一行同步）。
func normalizeComment(s *commentModel.SystemSetting) {
	if s.EmailNotify.SMTPPort <= 0 {
		s.EmailNotify.SMTPPort = 587
	}
}

// migratePasskeyFromLegacy 从旧 oauth2_setting 中读取曾经内联的 WebAuthn 字段。
func migratePasskeyFromLegacy(ctx context.Context, kv kvstore.Store) (settingModel.PasskeySetting, bool) {
	var result settingModel.PasskeySetting
	raw, err := kv.Get(ctx, commonModel.OAuth2SettingKey)
	if err != nil || strings.TrimSpace(raw) == "" {
		return result, false
	}
	var legacy struct {
		WebAuthnRPID           string   `json:"webauthn_rp_id"`
		WebAuthnAllowedOrigins []string `json:"webauthn_allowed_origins"`
	}
	if err := json.Unmarshal([]byte(raw), &legacy); err != nil {
		return result, false
	}
	result.WebAuthnRPID = strings.TrimSpace(legacy.WebAuthnRPID)
	result.WebAuthnAllowedOrigins = sanitizeURLList(legacy.WebAuthnAllowedOrigins)
	return result, result.WebAuthnRPID != "" || len(result.WebAuthnAllowedOrigins) > 0
}

// serverURLSeed 把 System.ServerURL 这一派生值同步进 server_url 便捷键（comment 等
// 只需 URL 的消费方读它，省去反序列化整个 SystemSetting）。复用 System.Default 保持单一来源。
type serverURLSeed struct{}

func (serverURLSeed) seed(ctx context.Context, kv kvstore.Store) error {
	if _, err := kv.Get(ctx, commonModel.ServerURLKey); err == nil {
		return nil
	} else if !errors.Is(err, kvstore.ErrNotFound) {
		return err
	}
	sys := System.Default()
	if System.Normalize != nil {
		System.Normalize(&sys)
	}
	return kv.Set(ctx, commonModel.ServerURLKey, sys.ServerURL)
}

// name_format 默认风格
const defaultNameFormatValue = "{userid:8}_{timestamp}_{hex:8}"
func defaultNameFormat(format string) string {
	if strings.TrimSpace(format) == "" {
		return defaultNameFormatValue
	}
	return strings.TrimSpace(format)
}

// stripScheme 去掉 endpoint 的 http(s):// 前缀，与历史读路径清洗一致。
func stripScheme(s string) string {
	return strings.TrimPrefix(strings.TrimPrefix(s, "http://"), "https://")
}

// sanitizeURLList TrimURL 每一项并剔除空串。
func sanitizeURLList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, v := range values {
		if trimmed := urlUtil.TrimURL(v); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
