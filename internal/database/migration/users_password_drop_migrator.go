// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package migration

import (
	"fmt"

	commonModel "github.com/lin-snow/ech0/internal/model/common"
	userModel "github.com/lin-snow/ech0/internal/model/user"
	"gorm.io/gorm"
)

// usersPasswordDropMigrator 删除 users 表遗留的 password 列。
//
// 本地密码已迁入 user_local_auth（见 userLocalAuthBackfillMigrator，务必排在其之后），
// 保留旧列意味着历史裸 MD5 仍残留在 users 表里 —— 删列才是安全升级的收口。
//
// SQLite 3.35+ 支持 ALTER TABLE ... DROP COLUMN；password 是无索引的普通列，可直接删。
// HasColumn 守卫使之天然幂等：列不存在（新库或已删）即跳过。
type usersPasswordDropMigrator struct{}

func NewUsersPasswordDropMigrator() Migrator {
	return &usersPasswordDropMigrator{}
}

func (m *usersPasswordDropMigrator) Name() string {
	return "users_password_drop_migrator"
}

func (m *usersPasswordDropMigrator) Key() string {
	return commonModel.UsersPasswordColumnDroppedKey
}

func (m *usersPasswordDropMigrator) CanRerun() bool {
	return false
}

func (m *usersPasswordDropMigrator) Migrate(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	if !db.Migrator().HasColumn(&userModel.User{}, "password") {
		return nil
	}
	return db.Exec(`ALTER TABLE users DROP COLUMN password`).Error
}
