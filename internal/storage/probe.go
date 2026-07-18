// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package storage

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/lin-snow/ech0/internal/config"
	settingModel "github.com/lin-snow/ech0/internal/model/setting"
	"github.com/lin-snow/ech0/pkg/virefs"
)

// errBucketRequired 在未填写 Bucket 时返回，给出比 SDK 底层报错更直观的提示。
var errBucketRequired = errors.New("S3 Bucket 不能为空")

// TestS3Connection 用给定 S3 设置做一次最小连通性探测（HeadBucket），验证 endpoint / 凭证 /
// bucket 是否真正可用。构造一次性 client 即用即弃，不触碰当前生效的存储配置；provider / region /
// 协议头的归一化复用与 buildS3FS 相同的逻辑，确保「测的就是会用的」那套参数。
func (m *Manager) TestS3Connection(ctx context.Context, setting settingModel.S3Setting) error {
	return probeS3(ctx, storageConfigFromSetting(setting, m.defaultCfg))
}

func probeS3(ctx context.Context, cfg config.StorageConfig) error {
	if cfg.BucketName == "" {
		return errBucketRequired
	}

	client, err := virefs.NewS3Client(ctx, virefsS3ConfigFromStorage(cfg))
	if err != nil {
		return err
	}

	// HeadBucket 是最轻量的探活：一次请求即可同时验证网络可达、凭证有效、bucket 存在且可访问，
	// 且不读写任何对象。
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(cfg.BucketName)})
	return err
}
