# Ech0 task runner — parallel to the Makefile for contributors who prefer
# `just` (https://github.com/casey/just), or who can't easily run GNU Make
# on Windows.
#
# Requires: just + bash. On Windows install Git for Windows (ships Git Bash)
# so the `bash` shell below is available.
#
# This file mirrors Makefile recipes 1:1. If you change a recipe here, mirror
# it in Makefile (and vice versa).

set shell := ["bash", "-cu"]

# --- Build metadata (resolved at parse time, same as Makefile) ---
VERSION       := `git describe --tags --always 2>/dev/null || echo unknown`
BUILD_TIME    := `date -u +%Y-%m-%dT%H:%M:%SZ`
BUILD_LABEL   := `echo "${BUILD_LABEL:-custom}"`

VERSION_PKG   := "github.com/lin-snow/ech0/internal/version"
LDFLAGS       := "-X " + VERSION_PKG + ".Commit=" + BUILD_LABEL + " -X " + VERSION_PKG + ".BuildTime=" + BUILD_TIME

# --- Docker (overridable via env: DOCKER_REGISTRY=foo just build-image) ---
GOHOSTOS        := `go env GOHOSTOS`
GOHOSTARCH      := `go env GOHOSTARCH`
DOCKER_REGISTRY := env_var_or_default("DOCKER_REGISTRY", "sn0wl1n")
IMAGE_NAME      := env_var_or_default("IMAGE_NAME", "ech0")
IMAGE_TAG       := env_var_or_default("IMAGE_TAG", "latest")
OS              := env_var_or_default("OS", GOHOSTOS)
ARCH            := env_var_or_default("ARCH", GOHOSTARCH)

# Default: list available recipes
default:
    @just --list

# Install Air (Go hot-reload tool) into $GOPATH/bin
air-install:
    go install github.com/air-verse/air@latest

# Run backend in serve mode
run:
    go run -ldflags "{{LDFLAGS}}" ./cmd/ech0 serve

# Build local binary with version/commit injected
build:
    go build -ldflags "{{LDFLAGS}}" -o ./bin/ech0 ./cmd/ech0

# Run backend with Air hot reload (auto-installs Air if missing)
dev:
    #!/usr/bin/env bash
    set -euo pipefail
    AIR_BIN="$(command -v air 2>/dev/null || echo "$(go env GOPATH)/bin/air")"
    if [ ! -x "$AIR_BIN" ]; then
        echo "air not found, installing..."
        just air-install
        AIR_BIN="$(go env GOPATH)/bin/air"
    fi
    "$AIR_BIN" -c .air.toml

# Run frontend dev server
web-dev:
    cd web && pnpm dev

# Backend fmt/lint + web format/lint + i18n checks (mandatory pre-PR)
check: dev-lint

dev-lint:
    bash scripts/check.sh

# Run golangci-lint checks
lint:
    golangci-lint run

# Run golangci-lint formatters
fmt:
    golangci-lint fmt

# Run Go tests
test:
    go test ./...

# Generate DI code via Wire
wire:
    go generate ./internal/di

# Verify Wire code is up-to-date (used by CI)
wire-check: wire
    git diff --exit-code -- internal/di/wire_gen.go

# Regenerate Swagger docs
swagger:
    swag init -g internal/server/server.go -o internal/swagger --parseInternal

# Add SPDX/Copyright headers to new .go/.ts/.vue files
spdx:
    node scripts/add-spdx-headers.mjs

# Fail if any source file is missing the SPDX header
spdx-check:
    node scripts/add-spdx-headers.mjs --check

# Bump internal/version.Version + sanity-check (does NOT commit/tag).
# See docs/dev/release-process.md for the full procedure.
# Usage: just bump 4.7.5
bump NEW_VERSION:
    #!/usr/bin/env bash
    set -euo pipefail
    SEMVER='^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$'
    if ! echo "{{NEW_VERSION}}" | grep -Eq "$SEMVER"; then
        echo "✘ '{{NEW_VERSION}}' is not valid semver (expected X.Y.Z[-prerelease])"
        exit 1
    fi
    if [ -n "$(git status --porcelain)" ]; then
        echo "✘ Working tree dirty — commit or stash first so the bump commit is clean."
        git status --short
        exit 1
    fi
    OLD_VERSION="$(grep -E '^[[:space:]]*Version[[:space:]]*=[[:space:]]*"' internal/version/version.go \
                    | head -n1 \
                    | sed -E 's/.*"([^"]+)".*/\1/')"
    if [ -z "$OLD_VERSION" ]; then
        echo "✘ Could not extract current Version from internal/version/version.go"
        exit 1
    fi
    if [ "$OLD_VERSION" = "{{NEW_VERSION}}" ]; then
        echo "✘ Version is already $OLD_VERSION — nothing to bump."
        exit 1
    fi
    echo "→ bumping $OLD_VERSION → {{NEW_VERSION}}"
    sed -i.bak -E "s/^([[:space:]]*Version[[:space:]]*=[[:space:]]*\")[^\"]+(\")/\\1{{NEW_VERSION}}\\2/" internal/version/version.go
    rm -f internal/version/version.go.bak
    echo "→ verifying go build still succeeds..."
    go build ./... >/dev/null || { echo "✘ go build failed after bump — reverting"; git checkout -- internal/version/version.go; exit 1; }
    echo ""
    echo "✓ Version bumped. Diff:"
    git --no-pager diff -- internal/version/version.go
    echo ""
    echo "Next steps (review the diff above, then run):"
    echo ""
    echo "  # 1. Update CHANGELOG.md: rename [Unreleased] → [{{NEW_VERSION}}] - $(date -u +%Y-%m-%d), open a new empty [Unreleased]"
    echo "  # 2. Commit + tag:"
    echo "       git commit -am 'chore(release): v{{NEW_VERSION}}'"
    echo "       git tag -a v{{NEW_VERSION}} -m 'Release v{{NEW_VERSION}}'"
    echo "  # 3. Push to trigger release workflow:"
    echo "       git push origin main"
    echo "       git push origin v{{NEW_VERSION}}"

# Build Docker image (override platform with OS=... ARCH=...)
build-image:
    @echo "Building image for platform: {{OS}}/{{ARCH}}"
    docker build --platform {{OS}}/{{ARCH}} \
        --build-arg TARGETOS={{OS}} \
        --build-arg TARGETARCH={{ARCH}} \
        --build-arg GIT_COMMIT={{GIT_COMMIT}} \
        --build-arg BUILD_TIME={{BUILD_TIME}} \
        -t {{DOCKER_REGISTRY}}/{{IMAGE_NAME}}:{{IMAGE_TAG}} -f docker/build.Dockerfile .

# Push Docker image
push-image:
    docker push {{DOCKER_REGISTRY}}/{{IMAGE_NAME}}:{{IMAGE_TAG}}
