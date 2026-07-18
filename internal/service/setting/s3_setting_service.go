// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package service

import (
	"context"
	"errors"
	"strings"
	"time"

	commonModel "github.com/lin-snow/ech0/internal/model/common"
	model "github.com/lin-snow/ech0/internal/model/setting"
	coreSetting "github.com/lin-snow/ech0/internal/setting"
	urlUtil "github.com/lin-snow/ech0/internal/util/url"
	"github.com/lin-snow/ech0/pkg/viewer"
)

// s3TestTimeout 是连通性探测的整体超时，避免坏 endpoint 把请求挂死。
const s3TestTimeout = 15 * time.Second

// GetS3Setting 获取 S3 存储设置。缺省值由 setting 引擎处理；脱敏（非管理员屏蔽敏感字段）
// 属请求态逻辑，留在此处。
func (settingService *SettingService) GetS3Setting(ctx context.Context, setting *model.S3Setting) error {
	userid := viewer.MustFromContext(ctx).UserID()
	v, err := coreSetting.Get(ctx, settingService.durableKV, coreSetting.S3)
	if err != nil {
		return err
	}
	*setting = v

	// 未登录直接脱敏。
	if userid == "" {
		maskS3Secrets(setting)
		return nil
	}
	user, err := settingService.commonService.CommonGetUserByUserId(ctx, userid)
	if err != nil {
		return err
	}
	if !user.IsAdmin {
		maskS3Secrets(setting)
	}
	return nil
}

func maskS3Secrets(setting *model.S3Setting) {
	setting.AccessKey = "******"
	setting.SecretKey = "******"
	setting.BucketName = "******"
	setting.Endpoint = "******"
}

// UpdateS3Setting 更新 S3 存储设置
func (settingService *SettingService) UpdateS3Setting(
	ctx context.Context,
	newSetting *model.S3SettingDto,
) error {
	userid := viewer.MustFromContext(ctx).UserID()
	user, err := settingService.commonService.CommonGetUserByUserId(ctx, userid)
	if err != nil {
		return err
	}
	if !user.IsAdmin {
		return errors.New(commonModel.NO_PERMISSION_DENIED)
	}

	oldRaw, _ := settingService.durableKV.Get(ctx, commonModel.S3SettingKey)
	var appliedSetting *model.S3Setting

	err = settingService.transactor.Run(ctx, func(ctx context.Context) error {
		s3Setting := normalizeS3SettingDto(newSetting)
		if err := coreSetting.Set(ctx, settingService.durableKV, coreSetting.S3, s3Setting); err != nil {
			return err
		}

		appliedSetting = &s3Setting
		return nil
	})
	if err != nil {
		return err
	}

	if settingService.storageManager != nil && appliedSetting != nil {
		if err := settingService.storageManager.ApplyS3Setting(*appliedSetting); err != nil {
			_ = settingService.transactor.Run(context.Background(), func(ctx context.Context) error {
				if strings.TrimSpace(oldRaw) == "" {
					return settingService.durableKV.Delete(ctx, commonModel.S3SettingKey)
				}
				return settingService.durableKV.Set(ctx, commonModel.S3SettingKey, oldRaw)
			})
			_ = settingService.storageManager.ReloadFromConfigAndDB(context.Background())
			return err
		}
	}

	return nil
}

// TestS3Connection 用提交的 S3 配置做一次连通性探测（不落库）。配置归一化与保存共用
// normalizeS3SettingDto，确保「测的就是会存的」；真正的 HeadBucket 探活在 storage 层完成。
func (settingService *SettingService) TestS3Connection(
	ctx context.Context,
	newSetting *model.S3SettingDto,
) error {
	userid := viewer.MustFromContext(ctx).UserID()
	user, err := settingService.commonService.CommonGetUserByUserId(ctx, userid)
	if err != nil {
		return err
	}
	if !user.IsAdmin {
		return errors.New(commonModel.NO_PERMISSION_DENIED)
	}
	if settingService.storageManager == nil {
		return errors.New("存储管理器不可用")
	}

	ctx, cancel := context.WithTimeout(ctx, s3TestTimeout)
	defer cancel()
	return settingService.storageManager.TestS3Connection(ctx, normalizeS3SettingDto(newSetting))
}

// normalizeS3SettingDto 把前端 DTO 规整为持久化用的 S3Setting：依据 endpoint 协议头推导 UseSSL、
// 去除 endpoint 协议头与 CDN 末尾斜杠、按 provider 补齐 region 默认值。UpdateS3Setting（保存）与
// TestS3Connection（连通性测试）共用，保证两条路径对同一份输入做完全一致的归一化。
func normalizeS3SettingDto(newSetting *model.S3SettingDto) model.S3Setting {
	// 检查endpoint是否为http(s)动态改变USE SSL
	useSSL := newSetting.UseSSL
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(newSetting.Endpoint)), "https://") {
		useSSL = true
	} else if strings.HasPrefix(strings.ToLower(strings.TrimSpace(newSetting.Endpoint)), "http://") {
		useSSL = false
	}

	// 去除Endpoint的协议头（http://或https://）
	endpoint := strings.TrimSpace(newSetting.Endpoint)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	cdnURL := strings.TrimSpace(newSetting.CDNURL)
	if cdnURL != "" {
		cdnURL = strings.TrimRight(cdnURL, "/")
	}

	s3Setting := model.S3Setting{
		Enable:       newSetting.Enable,
		Provider:     newSetting.Provider,
		Endpoint:     urlUtil.TrimURL(endpoint),
		AccessKey:    newSetting.AccessKey,
		SecretKey:    newSetting.SecretKey,
		BucketName:   newSetting.BucketName,
		Region:       strings.TrimSpace(newSetting.Region),
		UseSSL:       useSSL,
		CDNURL:       cdnURL,
		PathPrefix:   urlUtil.TrimURL(newSetting.PathPrefix),
		PublicRead:   newSetting.PublicRead,
		UsePathStyle: newSetting.UsePathStyle,
	}

	// 配置检查：按 provider 补齐 region 默认值
	switch s3Setting.Provider {
	case string(commonModel.R2):
		if s3Setting.Region == "" {
			s3Setting.Region = "auto"
		}
		s3Setting.UseSSL = true
	case string(commonModel.AWS):
		if s3Setting.Region == "" {
			s3Setting.Region = "us-east-1"
		}
	case string(commonModel.MINIO):
		if s3Setting.Region == "" {
			s3Setting.Region = "us-east-1"
		}
	case string(commonModel.OTHER):
		// 其他 S3 兼容厂商（Backblaze、Wasabi、Ceph 等）
		if s3Setting.Region == "" {
			s3Setting.Region = "auto"
		}
	default:
	}

	// path-style 开关仅对 other 生效：aws/minio/r2 的寻址方式由 virefs 预设决定，
	// 归零可避免切换 provider 后残留的 true 不可见地强制 path-style。
	if s3Setting.Provider != string(commonModel.OTHER) {
		s3Setting.UsePathStyle = false
	}

	return s3Setting
}
