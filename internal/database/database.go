// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package database

import (
	"errors"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	"github.com/lin-snow/ech0/internal/config"
	dbMigration "github.com/lin-snow/ech0/internal/database/migration"
	authModel "github.com/lin-snow/ech0/internal/model/auth"
	commentModel "github.com/lin-snow/ech0/internal/model/comment"
	commonModel "github.com/lin-snow/ech0/internal/model/common"
	connectModel "github.com/lin-snow/ech0/internal/model/connect"
	echoModel "github.com/lin-snow/ech0/internal/model/echo"
	embeddingModel "github.com/lin-snow/ech0/internal/model/embedding"
	fileModel "github.com/lin-snow/ech0/internal/model/file"
	jobModel "github.com/lin-snow/ech0/internal/model/job"
	settingModel "github.com/lin-snow/ech0/internal/model/setting"
	userModel "github.com/lin-snow/ech0/internal/model/user"
	visitorModel "github.com/lin-snow/ech0/internal/model/visitor"
	webhookModel "github.com/lin-snow/ech0/internal/model/webhook"
	util "github.com/lin-snow/ech0/internal/util/err"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 全局数据库连接变量
// var DB *gorm.DB

// 使用 atomic.Value 来存储 *gorm.DB，确保线程安全和支持热更新
var db atomic.Value // 用于存储 *gorm.DB

var writeLocked atomic.Bool

func GetDB() *gorm.DB {
	return db.Load().(*gorm.DB)
}

func SetDB(newDB *gorm.DB) {
	db.Store(newDB)
}

// func DBProvider() func() *gorm.DB {
// 	return GetDB
// }

// EnableWriteLock 启用写锁，阻止新的写操作
func EnableWriteLock() {
	writeLocked.Store(true)
}

// DisableWriteLock 关闭写锁，允许写操作
func DisableWriteLock() {
	writeLocked.Store(false)
}

// SetWriteLock 手动设置写锁状态
func SetWriteLock(enabled bool) {
	writeLocked.Store(enabled)
}

// IsWriteLocked 判断当前是否启用了写锁
func IsWriteLocked() bool {
	return writeLocked.Load()
}

func buildGormConfig(logLevel logger.LogLevel) *gorm.Config {
	return &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	}
}

// sqliteConnParams 是统一追加在 DSN 上的连接参数（mattn/go-sqlite3 形式）。
// 必须放在 DSN 而非 Open 后 Exec PRAGMA：busy_timeout / synchronous / txlock
// 都是单连接属性，只有 DSN 参数才能覆盖连接池里的每一条连接。
//   - _journal_mode=WAL：读写互不阻塞（持久化在库文件上）；
//   - _busy_timeout=5000：锁冲突时等待重试，而不是立刻报 database is locked；
//   - _synchronous=NORMAL：WAL 下的推荐档位，保证崩溃一致性；
//   - _txlock=immediate：写事务开始即持写锁，避免 deferred 事务升级写锁失败。
const sqliteConnParams = "_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_txlock=immediate"

// openSQLite 以统一的连接参数打开运行库。InitDatabase 与 HotChangeDatabase 共用，
// 保证热切换后的新连接与启动时行为一致。
func openSQLite(dbPath string, logLevel logger.LogLevel) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(dbPath+"?"+sqliteConnParams), buildGormConfig(logLevel))
}

// configLogLevel 按配置解析 GORM 日志级别。
func configLogLevel() logger.LogLevel {
	if config.Config().Database.LogMode == "release" {
		return logger.Silent
	}
	return logger.Error
}

// SnapshotTo 用 SQLite 官方在线备份语句 VACUUM INTO 把当前库写出一份一致性时点副本。
// 不阻塞并发读写，产出为独立单文件（不依赖 -wal/-shm 伴生文件），供快照导出打包。
// dstPath 指向的文件不能已存在（VACUUM INTO 语义）。
func SnapshotTo(dstPath string) error {
	return GetDB().Exec("VACUUM INTO ?", dstPath).Error
}

// InitDatabase 初始化数据库连接
func InitDatabase() {
	// 读取数据库类型和保存路径
	dbType := config.Config().Database.Type
	dbPath := config.Config().Database.Path

	dir := dbPath[:len(dbPath)-len("/ech0.db")] // 提取目录部分
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		util.HandlePanicError(&commonModel.ServerError{
			Msg: commonModel.CREATE_DB_PATH_PANIC,
			Err: err,
		})
	}

	if dbType == "sqlite" {
		// 注册 sqlite-vec 为进程级自动扩展：之后所有新建的 sqlite 连接（含热切换）
		// 都会自动加载 vec0 虚表能力，无需自定义 driver / ConnectHook。
		sqlite_vec.Auto()
		SQLiteDB, err := openSQLite(dbPath, configLogLevel())
		if err != nil {
			util.HandlePanicError(&commonModel.ServerError{
				Msg: commonModel.INIT_DATABASE_PANIC,
				Err: err,
			})
		}
		SetDB(SQLiteDB)
	}

	// 自动建表
	if err := MigrateDB(); err != nil {
		util.HandlePanicError(&commonModel.ServerError{
			Msg: commonModel.MIGRATE_DB_PANIC,
			Err: err,
		})
	}

	dbMigration.Migrate(
		GetDB(),
		dbMigration.WithStopOnError(),
		dbMigration.WithMigrators(
			dbMigration.NewLegacyTimeNormalizerMigrator(dbMigration.DefaultLegacySourceTimezone),
			dbMigration.NewStorageTimeSanitizeMigrator(),
			dbMigration.NewStorageTimeValidateMigrator(),
			dbMigration.NewStorageTimeUnixMigrator(),
			dbMigration.NewStorageTimeSchemaRebuildMigrator(),
			dbMigration.NewOAuthBindingsDropMigrator(),
			dbMigration.NewLegacyInboxesDropMigrator(),
			dbMigration.NewAgentProtocolCollapseMigrator(),
			dbMigration.NewAgentSettingProtocolRenameMigrator(),
			// 本地密码迁入 user_local_auth：回填必须先于删列。
			dbMigration.NewUserLocalAuthBackfillMigrator(),
			dbMigration.NewUsersPasswordDropMigrator(),
			dbMigration.NewEchoExtensionOrphansMigrator(),
		),
	)
}

// MigrateDB 执行数据库迁移
func MigrateDB() error {
	models := []interface{}{
		&userModel.User{},
		&userModel.UserLocalAuth{},
		&userModel.UserExternalIdentity{},
		&userModel.WebAuthnCredential{},
		&echoModel.Echo{},
		&echoModel.EchoExtension{},
		&embeddingModel.EchoEmbedding{},
		&fileModel.File{},
		&fileModel.EchoFile{},
		&fileModel.TempFile{},
		&commonModel.KeyValue{},
		&connectModel.Connected{},
		&echoModel.Tag{},
		&echoModel.EchoTag{},
		&commentModel.Comment{},
		&webhookModel.Webhook{},
		&jobModel.Job{},
		&settingModel.AccessTokenSetting{},
		&authModel.Passkey{},
		&visitorModel.DailyStat{},
	}

	return GetDB().AutoMigrate(
		models...,
	)
}

// HotChangeDatabase 热切换数据库连接
func HotChangeDatabase(newDBPath string) error {
	// 获取当前数据库连接
	oldDB := GetDB()

	// 彻底关闭旧连接
	if oldDB != nil {
		if err := CloseDatabaseFully(oldDB); err != nil {
			return err
		}
	}

	// 打开新连接
	newDB, err := openSQLite(newDBPath, configLogLevel())
	if err != nil {
		return err
	}

	SetDB(newDB)
	return nil
}

// CloseDatabaseFully 彻底关闭数据库连接，释放资源
func CloseDatabaseFully(db *gorm.DB) error {
	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		if err := sqlDB.Close(); err != nil {
			return err
		}
		SetDB(nil)

		// 强制 GC 回收
		runtime.GC()
		time.Sleep(100 * time.Millisecond)

		return nil
	}

	return errors.New(commonModel.DATABASE_CLOSE_FAILED)
}
