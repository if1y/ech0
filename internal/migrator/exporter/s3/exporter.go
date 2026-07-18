// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

// Package s3 是「导出到对象存储」的 Exporter 适配器:产出 Snapshot 后在后台上传到 S3,同时保留
// 本地产物(故下载仍可取回)。S3 上传为尽力而为,移出作业关键路径放到后台跑:本地产物一落盘作业即
// 判完成、下载立即可用,上传失败仅记日志、不影响本地导出成功,也不阻塞下载。
package s3

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/lin-snow/ech0/internal/database"
	"github.com/lin-snow/ech0/internal/migrator/snapshot"
	"github.com/lin-snow/ech0/internal/migrator/spec"
	migratorModel "github.com/lin-snow/ech0/internal/model/migrator"
	"github.com/lin-snow/ech0/internal/storage"
	logUtil "github.com/lin-snow/ech0/pkg/log"
)

const uploadTimeout = 60 * time.Minute

type Exporter struct {
	storageManager *storage.Manager
}

func New(storageManager *storage.Manager) *Exporter {
	return &Exporter{storageManager: storageManager}
}

func (e *Exporter) Export(_ context.Context, req spec.ExportRequest) (spec.ExportResult, error) {
	emit(req, migratorModel.ExportPhasePacking)
	// 导出发生在运行中的实例上,必须用 VACUUM INTO 产出的一致性副本代替实时库文件。
	path, fileName, err := snapshot.Create(snapshot.WithConsistentDB(database.SnapshotTo))
	if err != nil {
		return spec.ExportResult{}, err
	}

	var size int64
	if info, statErr := os.Stat(path); statErr == nil {
		size = info.Size()
	}

	// 本地产物已落盘:作业即刻判完成、下载立即可用。S3 上传是尽力而为的副作用,移出关键路径
	// 放到后台跑(uploadToS3 自带 background context + 超时,与作业生命周期解耦),失败仅记日志。
	go e.uploadToS3(path, fileName)

	emit(req, migratorModel.ExportPhaseCompleted)
	return spec.ExportResult{ArtifactPath: path, FileName: fileName, Size: size}, nil
}

// uploadToS3 把产物上传到 S3;未启用对象存储或上传失败均不影响本地导出成功,仅记日志。
func (e *Exporter) uploadToS3(artifactPath, fileName string) {
	if e.storageManager == nil {
		return
	}
	selector := e.storageManager.GetSelector()
	if selector == nil || !selector.ObjectEnabled() {
		return
	}

	uploadCtx, cancel := context.WithTimeout(context.Background(), uploadTimeout)
	defer cancel()

	cfg := e.storageManager.GetStorageConfig(uploadCtx)
	if err := snapshot.UploadToS3(uploadCtx, artifactPath, fileName, cfg); err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			logUtil.GetLogger().Warn("Failed to upload snapshot to S3: upload timeout reached",
				slog.Duration("timeout", uploadTimeout), logUtil.Err(err))
		case errors.Is(err, context.Canceled):
			logUtil.GetLogger().Warn("Failed to upload snapshot to S3: upload context canceled", logUtil.Err(err))
		default:
			logUtil.GetLogger().Warn("Failed to upload snapshot to S3", logUtil.Err(err))
		}
	}
}

func emit(req spec.ExportRequest, phase string) {
	if req.UpdateProgress != nil {
		req.UpdateProgress(spec.ExportProgress{CurrentPhase: phase})
	}
}
