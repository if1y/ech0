// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package storage

import (
	"context"
	"log/slog"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/wire"
	"github.com/lin-snow/ech0/internal/config"
	"github.com/lin-snow/ech0/internal/kvstore"
	logUtil "github.com/lin-snow/ech0/pkg/log"
	"github.com/lin-snow/ech0/pkg/virefs"
)

func ProvideStorageManager(durableKV kvstore.Store) *Manager { return NewStorageManager(durableKV) }

var (
	ManagerSet  = wire.NewSet(ProvideStorageManager)
	ProviderSet = wire.NewSet(ManagerSet)
)

// NewFS builds a virefs.FS based on the given StorageConfig.
// File classification (images/, audios/, etc.) is handled by VireFS Schema.
func NewFS(cfg config.StorageConfig) virefs.FS {
	schema := NewFileSchema()
	if cfg.ObjectEnabled {
		return buildS3FS(cfg, schema)
	}
	return buildLocalFS(cfg, schema)
}

// NewURLResolver builds a URLResolver based on the given StorageConfig.
// It applies schema.Resolve internally so callers just pass flat keys.
func NewURLResolver(cfg config.StorageConfig) URLResolver {
	schema := NewFileSchema()
	if cfg.ObjectEnabled {
		return buildS3URLResolver(cfg, schema)
	}
	return buildLocalURLResolver(schema)
}

func buildLocalFS(cfg config.StorageConfig, schema *virefs.Schema) virefs.FS {
	root := cfg.DataRoot
	if root == "" {
		root = "data/files"
	}
	fs, err := virefs.NewLocalFS(root,
		virefs.WithCreateRoot(),
		virefs.WithAtomicWrite(),
		virefs.WithLocalKeyFunc(schema.Resolve),
	)
	if err != nil {
		logUtil.Warn("create local fs failed, fallback to defaults", slog.String("module", "storage"), logUtil.Err(err))
		fs, _ = virefs.NewLocalFS("data/files",
			virefs.WithCreateRoot(),
			virefs.WithAtomicWrite(),
			virefs.WithLocalKeyFunc(schema.Resolve),
		)
	}
	return fs
}

func buildLocalURLResolver(schema *virefs.Schema) URLResolver {
	pathResolver := buildLocalPathURLResolver()
	return func(key string) string {
		return pathResolver(schema.Resolve(key))
	}
}

func buildLocalPathURLResolver() URLResolver {
	return func(path string) string {
		clean := strings.Trim(strings.TrimSpace(path), "/")
		if clean == "" {
			return "/api/files"
		}
		return "/api/files/" + clean
	}
}

func buildS3FS(cfg config.StorageConfig, schema *virefs.Schema) virefs.FS {
	var opts []virefs.ObjectOption
	if cfg.PathPrefix != "" {
		opts = append(opts, virefs.WithPrefix(strings.Trim(cfg.PathPrefix, "/")+"/"))
	}
	opts = append(opts, virefs.WithObjectKeyFunc(schema.Resolve))

	fs, err := virefs.NewObjectFSFromConfig(context.Background(), virefsS3ConfigFromStorage(cfg), opts...)
	if err != nil {
		logUtil.Warn("create s3 fs failed, fallback to local", slog.String("module", "storage"), logUtil.Err(err))
		return buildLocalFS(cfg, schema)
	}
	return fs
}

// virefsS3ConfigFromStorage 把操作用 StorageConfig 归一化为 virefs.S3Config，
// buildS3FS / probeS3 / buildOptionalObjectFSAndResolver 共用同一套参数，
// 保证「测的就是会用的」。
func virefsS3ConfigFromStorage(cfg config.StorageConfig) *virefs.S3Config {
	s3cfg := &virefs.S3Config{
		Provider:  mapProvider(cfg.Provider),
		Endpoint:  normalizeEndpoint(cfg.Endpoint, cfg.UseSSL),
		Region:    resolveObjectRegion(cfg.Provider, cfg.Region),
		Bucket:    cfg.BucketName,
		AccessKey: cfg.AccessKey,
		SecretKey: cfg.SecretKey,
	}
	// 仅在 true 时设指针：virefs 只在 UsePathStyle 为 nil 时应用 minio/r2 的
	// path-style 预设，无条件传 aws.Bool(false) 会把预设击穿。
	if cfg.UsePathStyle {
		s3cfg.UsePathStyle = aws.Bool(true)
	}
	return s3cfg
}

func buildS3URLResolver(cfg config.StorageConfig, schema *virefs.Schema) URLResolver {
	pathResolver := buildS3PathURLResolver(cfg)
	return func(key string) string {
		return pathResolver(schema.Resolve(key))
	}
}

func buildS3PathURLResolver(cfg config.StorageConfig) URLResolver {
	prefix := ""
	if cfg.PathPrefix != "" {
		prefix = strings.Trim(cfg.PathPrefix, "/") + "/"
	}

	cdnURL := strings.TrimSpace(cfg.CDNURL)
	if cdnURL != "" {
		if !strings.HasPrefix(strings.ToLower(cdnURL), "http://") &&
			!strings.HasPrefix(strings.ToLower(cdnURL), "https://") {
			protocol := "http"
			if cfg.UseSSL {
				protocol = "https"
			}
			cdnURL = protocol + "://" + cdnURL
		}
		cdnURL = strings.TrimRight(cdnURL, "/")
		return func(path string) string {
			clean := strings.Trim(strings.TrimSpace(path), "/")
			return cdnURL + "/" + prefix + clean
		}
	}

	endpoint := normalizeEndpoint(cfg.Endpoint, cfg.UseSSL)
	baseURL := strings.TrimRight(endpoint, "/") + "/" + cfg.BucketName
	return func(path string) string {
		clean := strings.Trim(strings.TrimSpace(path), "/")
		return baseURL + "/" + prefix + clean
	}
}

func normalizeEndpoint(endpoint string, useSSL bool) string {
	if endpoint == "" {
		return endpoint
	}
	lower := strings.ToLower(endpoint)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return endpoint
	}
	if useSSL {
		return "https://" + endpoint
	}
	return "http://" + endpoint
}

func mapProvider(raw string) virefs.Provider {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "minio":
		return virefs.ProviderMinIO
	case "r2":
		return virefs.ProviderR2
	default:
		return virefs.ProviderAWS
	}
}

func resolveObjectRegion(providerRaw string, regionRaw string) string {
	region := strings.TrimSpace(regionRaw)
	if region != "" {
		return region
	}
	switch strings.ToLower(strings.TrimSpace(providerRaw)) {
	case "r2", "other":
		return "auto"
	default:
		return "us-east-1"
	}
}
