// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package migration

import (
	"fmt"

	commonModel "github.com/lin-snow/ech0/internal/model/common"
	userModel "github.com/lin-snow/ech0/internal/model/user"
	"gorm.io/gorm"
)

// userLocalAuthBackfillMigrator 把存量的本地密码从 users.password（历史裸 MD5）
// 回填进独立的 user_local_auth 表，算法标记为 md5，等待下次登录时惰性升级为 bcrypt。
//
// 幂等双保险：
//   - 框架级：CanRerun()=false + 标记键，成功后写 KeyValue，重启即跳过；
//   - 语句级：user_local_auth.user_id 是主键，INSERT OR IGNORE 不会覆盖用户已改过的哈希。
//
// 新库（users 无 password 列）直接跳过——没有任何存量可回填。
type userLocalAuthBackfillMigrator struct{}

func NewUserLocalAuthBackfillMigrator() Migrator {
	return &userLocalAuthBackfillMigrator{}
}

func (m *userLocalAuthBackfillMigrator) Name() string {
	return "user_local_auth_backfill_migrator"
}

func (m *userLocalAuthBackfillMigrator) Key() string {
	return commonModel.UserLocalAuthBackfilledKey
}

func (m *userLocalAuthBackfillMigrator) CanRerun() bool {
	return false
}

func (m *userLocalAuthBackfillMigrator) Migrate(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	// 新库或已删列：无 users.password 列，无存量可回填。
	if !db.Migrator().HasColumn(&userModel.User{}, "password") {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// user_id 主键 → INSERT OR IGNORE 保证幂等；仅回填有非空密码、且尚无本地认证行的用户。
		return tx.Exec(`
			INSERT OR IGNORE INTO user_local_auth (user_id, password_hash, password_algo, updated_at)
			SELECT id, password, 'md5', CAST(strftime('%s','now') AS INTEGER)
			FROM users
			WHERE password IS NOT NULL AND password <> ''
		`).Error
	})
}
