// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

// Package fs 是「导出到本地目录」的 Exporter 适配器:把当前实例产出为一个 Snapshot
// (data/ 的 zip),落在本地快照目录。与导入侧的来源适配器对称。
package fs

import (
	"context"
	"os"

	"github.com/lin-snow/ech0/internal/database"
	"github.com/lin-snow/ech0/internal/migrator/snapshot"
	"github.com/lin-snow/ech0/internal/migrator/spec"
	migratorModel "github.com/lin-snow/ech0/internal/model/migrator"
)

type Exporter struct{}

func New() *Exporter {
	return &Exporter{}
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

	emit(req, migratorModel.ExportPhaseCompleted)
	return spec.ExportResult{ArtifactPath: path, FileName: fileName, Size: size}, nil
}

func emit(req spec.ExportRequest, phase string) {
	if req.UpdateProgress != nil {
		req.UpdateProgress(spec.ExportProgress{CurrentPhase: phase})
	}
}
