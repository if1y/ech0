// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package migration

import (
	"fmt"

	commonModel "github.com/lin-snow/ech0/internal/model/common"
	"gorm.io/gorm"
)

type echoExtensionOrphansMigrator struct{}

// NewEchoExtensionOrphansMigrator 清理历史遗留的 echo_extensions 孤儿行:
// 旧版删除 echo 时未手动级联删除 extension,而 schema 里的 ON DELETE CASCADE
// 因 SQLite 连接默认不开 foreign_keys 从未生效,孤儿行会静默累积。
func NewEchoExtensionOrphansMigrator() Migrator {
	return &echoExtensionOrphansMigrator{}
}

func (m *echoExtensionOrphansMigrator) Name() string {
	return "echo_extension_orphans_migrator"
}

func (m *echoExtensionOrphansMigrator) Key() string {
	return commonModel.EchoExtensionOrphansCleanedKey
}

func (m *echoExtensionOrphansMigrator) CanRerun() bool {
	return false
}

func (m *echoExtensionOrphansMigrator) Migrate(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	return db.Exec(
		`DELETE FROM echo_extensions WHERE echo_id NOT IN (SELECT id FROM echos)`,
	).Error
}
