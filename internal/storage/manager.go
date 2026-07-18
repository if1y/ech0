// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package storage

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/lin-snow/ech0/internal/config"
	"github.com/lin-snow/ech0/internal/kvstore"
	fileModel "github.com/lin-snow/ech0/internal/model/file"
	settingModel "github.com/lin-snow/ech0/internal/model/setting"
	coreSetting "github.com/lin-snow/ech0/internal/setting"
)

type Manager struct {
	mu         sync.RWMutex
	defaultCfg config.StorageConfig
	durableKV  kvstore.Store
	selector   *StorageSelector
}

func NewStorageManager(durableKV kvstore.Store) *Manager {
	defaultCfg := config.Config().Storage
	m := &Manager{
		defaultCfg: defaultCfg,
		durableKV:  durableKV,
		selector:   NewStorageSelector(defaultCfg),
	}
	_ = m.ReloadFromConfigAndDB(context.Background())
	// Publish the live URL resolver to the File model so reads recompute URLs
	// from the current config instead of trusting the write-time snapshot.
	fileModel.RegisterURLResolver(m.ResolveURL)
	return m
}

// NewStorageManagerForTest builds a Manager backed purely by local storage rooted
// at dataRoot, with object storage off and no DB. It exists so tests can exercise
// the file service against an isolated temp dir (t.TempDir()) instead of the
// config-derived ./data/files — Manager's fields are unexported, so an external
// test helper cannot assemble one without this seam. It performs no global side
// effects (does not register the URL resolver) to keep parallel test binaries clean.
func NewStorageManagerForTest(dataRoot string) *Manager {
	cfg := config.StorageConfig{DataRoot: dataRoot}
	return &Manager{
		defaultCfg: cfg,
		durableKV:  nil,
		selector:   NewStorageSelector(cfg),
	}
}

func (m *Manager) GetSelector() *StorageSelector {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.selector
}

// ResolveURL builds a file's public URL from its stored storage type + key
// using the current selector. The File model's AfterFind hook calls this to
// refresh URL snapshots on read; it always reflects the latest CDN/S3 config.
func (m *Manager) ResolveURL(storageType, key string) string {
	return m.GetSelector().ResolveURL(StorageType(storageType), key)
}

// GetStorageConfig returns the current effective storage configuration.
func (m *Manager) GetStorageConfig(ctx context.Context) config.StorageConfig {
	return m.resolveStorageConfig(ctx)
}

func (m *Manager) ReloadFromConfigAndDB(ctx context.Context) error {
	return m.replaceSelector(m.resolveStorageConfig(ctx))
}

func (m *Manager) ApplyS3Setting(setting settingModel.S3Setting) error {
	return m.replaceSelector(storageConfigFromSetting(setting, m.defaultCfg))
}

func (m *Manager) replaceSelector(cfg config.StorageConfig) error {
	selector := NewStorageSelector(cfg)
	if cfg.ObjectEnabled && !selector.ObjectEnabled() {
		return errors.New("object storage enabled but initialization failed")
	}
	m.mu.Lock()
	m.selector = selector
	m.mu.Unlock()
	return nil
}

// resolveStorageConfig assembles the config the selector runs on: S3 fields come
// from the panel-managed KV (via the setting engine), while local-only fields
// (DataRoot) come from config — the panel does not manage those.
func (m *Manager) resolveStorageConfig(ctx context.Context) config.StorageConfig {
	return storageConfigFromSetting(m.currentS3Setting(ctx), m.defaultCfg)
}

// currentS3Setting reads the S3 setting through the setting engine. setting.Get
// returns a config-derived, normalized default on any miss or failure (absent
// key, backend error, or unparseable value), so ignoring the error here always
// yields a sane setting. durableKV may be nil in unwired contexts (tests/CLI),
// in which case the config-derived default is used directly.
func (m *Manager) currentS3Setting(ctx context.Context) settingModel.S3Setting {
	if m.durableKV == nil {
		return coreSetting.S3.Default()
	}
	s3, _ := coreSetting.Get(ctx, m.durableKV, coreSetting.S3)
	return s3
}

// storageConfigFromSetting maps the panel-managed S3 setting onto the operational
// storage config, carrying local-only fields (DataRoot) over from defaultCfg.
//
// S3 fields are taken verbatim from the setting — there is NO per-field fallback
// to config. In this project env is not a user-facing config surface (only the
// JWT secret is set via env); config exists to seed internal/setting and to
// supply the few fields the panel does not manage (DataRoot). The KV s3_setting
// is therefore the single source of truth for S3, mirroring how the rest of the
// app reads settings via setting.Get(setting.S3). When the key is absent the
// setting engine already substitutes a config-derived default, so a fresh /
// unconfigured install still resolves to sane defaults (object storage off).
func storageConfigFromSetting(s3 settingModel.S3Setting, defaultCfg config.StorageConfig) config.StorageConfig {
	cfg := defaultCfg
	cfg.DataRoot = strings.TrimSpace(defaultCfg.DataRoot)
	if cfg.DataRoot == "" {
		cfg.DataRoot = "data/files"
	}

	cfg.ObjectEnabled = s3.Enable
	cfg.Provider = strings.TrimSpace(s3.Provider)
	cfg.Endpoint = trimEndpoint(s3.Endpoint)
	cfg.AccessKey = strings.TrimSpace(s3.AccessKey)
	cfg.SecretKey = strings.TrimSpace(s3.SecretKey)
	cfg.BucketName = strings.TrimSpace(s3.BucketName)
	cfg.Region = strings.TrimSpace(s3.Region)
	cfg.CDNURL = strings.TrimRight(strings.TrimSpace(s3.CDNURL), "/")
	cfg.PathPrefix = strings.Trim(strings.TrimSpace(s3.PathPrefix), "/")
	cfg.UseSSL = s3.UseSSL
	cfg.UsePathStyle = s3.UsePathStyle

	return cfg
}

func trimEndpoint(endpoint string) string {
	e := strings.TrimSpace(endpoint)
	e = strings.TrimPrefix(e, "http://")
	e = strings.TrimPrefix(e, "https://")
	return e
}
