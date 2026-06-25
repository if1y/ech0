// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package service

import (
	"context"
	"errors"
	"strings"

	i18nUtil "github.com/lin-snow/ech0/internal/i18n"
	commonModel "github.com/lin-snow/ech0/internal/model/common"
	model "github.com/lin-snow/ech0/internal/model/setting"
	coreSetting "github.com/lin-snow/ech0/internal/setting"
	logUtil "github.com/lin-snow/ech0/internal/util/log"
	urlUtil "github.com/lin-snow/ech0/internal/util/url"
	"github.com/lin-snow/ech0/pkg/viewer"
	"go.uber.org/zap"
)

// GetSetting 获取系统设置。缺省值/归一化统一由 setting 引擎处理，启动 seeder 已落库，
// 故这里只是一次读取。
func (settingService *SettingService) GetSetting(setting *model.SystemSetting) error {
	v, err := coreSetting.Get(context.Background(), settingService.durableKV, coreSetting.System)
	if err != nil {
		return err
	}
	*setting = v
	return nil
}

// BootstrapDefaultLocale 在首次部署（站长初始化）时把部署者语言写入站点默认。
// 启动 seeder 已落库 system_settings，故此处改为「就地更新」：仅当 DefaultLocale 仍为内置
// 默认（未被站长改过）时才覆盖，避免踩掉后续手动选择。入参为空/解析失败/等于默认时跳过。
func (settingService *SettingService) BootstrapDefaultLocale(
	ctx context.Context,
	locale string,
) error {
	resolved := strings.TrimSpace(locale)
	if resolved == "" {
		return nil
	}
	resolved = i18nUtil.ResolveLocale(resolved)
	if resolved == "" || resolved == string(commonModel.DefaultLocale) {
		return nil
	}

	return settingService.transactor.Run(ctx, func(ctx context.Context) error {
		current, err := coreSetting.Get(ctx, settingService.durableKV, coreSetting.System)
		if err != nil {
			return err
		}
		// 站长已手动设过非默认 locale 时不覆盖。
		if i18nUtil.ResolveLocale(current.DefaultLocale) != string(commonModel.DefaultLocale) {
			return nil
		}
		current.DefaultLocale = resolved
		return coreSetting.Set(ctx, settingService.durableKV, coreSetting.System, current)
	})
}

// UpdateSetting 更新设置
func (settingService *SettingService) UpdateSetting(
	ctx context.Context,
	newSetting *model.SystemSettingDto,
) error {
	userid := viewer.MustFromContext(ctx).UserID()
	serverLogoChanged := false
	if newSetting != nil {
		var current model.SystemSetting
		if err := settingService.GetSetting(&current); err == nil {
			serverLogoChanged = strings.TrimSpace(current.ServerLogo) != strings.TrimSpace(newSetting.ServerLogo)
		}
	}
	if err := settingService.transactor.Run(ctx, func(ctx context.Context) error {
		user, err := settingService.commonService.CommonGetUserByUserId(ctx, userid)
		if err != nil {
			return err
		}
		if !user.IsAdmin {
			return errors.New(commonModel.NO_PERMISSION_DENIED)
		}

		var setting model.SystemSetting
		setting.SiteTitle = newSetting.SiteTitle
		setting.ServerLogo = newSetting.ServerLogo
		setting.ServerName = newSetting.ServerName
		setting.ServerURL = urlUtil.TrimURL(newSetting.ServerURL)
		setting.LoaderImageURL = newSetting.LoaderImageURL
		setting.LoaderBrandText = newSetting.LoaderBrandText
		setting.LoaderSlogan = newSetting.LoaderSlogan
		setting.AllowRegister = newSetting.AllowRegister
		setting.DefaultLocale = i18nUtil.ResolveLocale(newSetting.DefaultLocale)
		setting.ICPNumber = newSetting.ICPNumber
		setting.FooterContent = newSetting.FooterContent
		setting.FooterLink = urlUtil.TrimURL(newSetting.FooterLink)
		setting.MetingAPI = urlUtil.TrimURL(newSetting.MetingAPI)
		setting.CustomCSS = newSetting.CustomCSS
		setting.CustomJS = newSetting.CustomJS

		if err := coreSetting.Set(ctx, settingService.durableKV, coreSetting.System, setting); err != nil {
			return err
		}

		// 同步派生的 ServerURL 便捷键。
		if err := settingService.durableKV.Set(ctx, commonModel.ServerURLKey, setting.ServerURL); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}
	if serverLogoChanged && strings.TrimSpace(newSetting.ServerLogoFileID) != "" {
		if err := settingService.fileService.ConfirmTempFiles(ctx, []string{newSetting.ServerLogoFileID}); err != nil {
			logUtil.GetLogger().Warn("confirm temp server logo file failed", zap.Error(err))
		}
	}
	return nil
}
