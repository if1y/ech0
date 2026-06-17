// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package storage

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const DefaultNameFormat = "{userid:8}_{timestamp}_{hex:8}"

var nameFormatTokenPattern = regexp.MustCompile(`\{([a-z]+)(?::(\d+))?\}`)

func compactUserID(userID string) string {
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return "anon"
	}
	uid = strings.ReplaceAll(uid, "-", "")
	return strings.ToLower(uid)
}

// RandomKeyGenerator creates keys from a configurable name format.
type RandomKeyGenerator struct {
	Format      string
	RandSource  io.Reader
	SuffixBytes int
	Now         func() time.Time
}

func NewRandomKeyGenerator(format string) *RandomKeyGenerator {
	return &RandomKeyGenerator{
		Format:      normalizeNameFormat(format),
		RandSource:  rand.Reader,
		SuffixBytes: 4,
		Now:         time.Now,
	}
}

func (g *RandomKeyGenerator) GenerateKey(_ Category, userID string, originalFilename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(originalFilename)))
	if ext == "" {
		ext = ".bin"
	}
	baseName := strings.TrimSuffix(filepath.Base(strings.TrimSpace(originalFilename)), filepath.Ext(strings.TrimSpace(originalFilename)))
	baseName = sanitizeNameSegment(baseName)
	if baseName == "" {
		baseName = "file"
	}
	formatted, err := g.renderNameFormat(userID, baseName)
	if err != nil {
		return "", err
	}
	formatted = sanitizeNameSegment(formatted)
	if formatted == "" {
		formatted = sanitizeNameSegment(compactUserID(userID))
	}
	if formatted == "" {
		formatted = "file"
	}
	return strings.ToLower(formatted) + ext, nil
}

func normalizeNameFormat(format string) string {
	if strings.TrimSpace(format) == "" {
		return DefaultNameFormat
	}
	return strings.TrimSpace(format)
}

func (g *RandomKeyGenerator) renderNameFormat(userID string, filename string) (string, error) {
	format := normalizeNameFormat(g.Format)
	nowFn := g.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn().UTC()
	uid := compactUserID(userID)
	var renderErr error
	result := nameFormatTokenPattern.ReplaceAllStringFunc(format, func(token string) string {
		if renderErr != nil {
			return ""
		}
		matches := nameFormatTokenPattern.FindStringSubmatch(token)
		if len(matches) != 3 {
			return token
		}
		name := matches[1]
		width := parseTokenWidth(matches[2])
		switch name {
		case "year":
			return now.Format("2006")
		case "month":
			return now.Format("01")
		case "day":
			return now.Format("02")
		case "hour":
			return now.Format("15")
		case "minute":
			return now.Format("04")
		case "second":
			return now.Format("05")
		case "millisecond":
			return fmt.Sprintf("%03d", now.Nanosecond()/int(time.Millisecond))
		case "timestamp":
			return strconv.FormatInt(now.Unix(), 10)
		case "filename":
			return truncateSegment(filename, width)
		case "userid":
			return truncateSegment(uid, width)
		case "hex":
			hexLen := width
			if hexLen <= 0 {
				hexLen = g.defaultHexLength()
			}
			randValue, err := g.randomHex(hexLen)
			if err != nil {
				renderErr = err
				return ""
			}
			return randValue
		default:
			return token
		}
	})
	if renderErr != nil {
		return "", renderErr
	}
	return result, nil
}

func parseTokenWidth(raw string) int {
	if strings.TrimSpace(raw) == "" {
		return 0
	}
	width, err := strconv.Atoi(raw)
	if err != nil || width <= 0 {
		return 0
	}
	return width
}

func truncateSegment(value string, width int) string {
	value = sanitizeNameSegment(value)
	if width > 0 && len(value) > width {
		return value[:width]
	}
	return value
}

func sanitizeNameSegment(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "_")
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.Trim(value, ".")
	return value
}

func (g *RandomKeyGenerator) defaultHexLength() int {
	size := g.SuffixBytes
	if size <= 0 {
		size = 4
	}
	return size * 2
}

func (g *RandomKeyGenerator) randomHex(length int) (string, error) {
	if length <= 0 {
		length = g.defaultHexLength()
	}
	byteLen := (length + 1) / 2
	src := g.RandSource
	if src == nil {
		src = rand.Reader
	}
	randPart := make([]byte, byteLen)
	if _, err := io.ReadFull(src, randPart); err != nil {
		return "", err
	}
	encoded := hex.EncodeToString(randPart)
	if len(encoded) > length {
		encoded = encoded[:length]
	}
	return encoded, nil
}

// StaticKeyGenerator produces a fixed key, useful for singleton files
// like background music.
type StaticKeyGenerator struct {
	Name string
}

func (g *StaticKeyGenerator) GenerateKey(_ Category, _ string, _ string) (string, error) {
	name := strings.TrimSpace(g.Name)
	if name == "" {
		return "", fmt.Errorf("static key name is empty")
	}
	return name, nil
}
