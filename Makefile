# Variables
GOHOSTOS:=$(shell go env GOHOSTOS)
GOHOSTARCH:=$(shell go env GOHOSTARCH)
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always 2>/dev/null || echo unknown)
BUILD_TIME=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
# 构建标识，默认 "custom"。构建时可通过 BUILD_LABEL=xxx make build 覆盖。
BUILD_LABEL?=custom

# Inject build metadata into the binary. Commit defaults to "custom" to avoid
# leaking the real git hash; override via BUILD_LABEL env var.
VERSION_PKG=github.com/lin-snow/ech0/internal/version
LDFLAGS=-X $(VERSION_PKG).Commit=$(BUILD_LABEL) -X $(VERSION_PKG).BuildTime=$(BUILD_TIME)

# Docker variables
DOCKER_REGISTRY?=sn0wl1n
IMAGE_NAME?=ech0
IMAGE_TAG?=latest
OS?=$(if $(GOHOSTOS),$(GOHOSTOS),linux)
ARCH?=$(if $(GOHOSTARCH),$(GOHOSTARCH),amd64)

.PHONY: help air-install run dev web-dev check dev-lint lint fmt test test-race test-cover mocks mocks-check wire wire-check openapi openapi-check spdx spdx-check build bump build-image push-image

# Semver pattern: X.Y.Z, optionally followed by a -prerelease suffix.
# (escape $ so make doesn't expand, then escape $$ to a literal $ in shell)
SEMVER_PATTERN := ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$$

AIR_BIN := $(shell command -v air 2>/dev/null || echo "$(GOPATH)/bin/air")

# mockery 仅作代码生成器，不进 go.mod（用 go run 固定版本调用），保持模块图精简。
# 版本 pin 死，保证任何机器/CI 生成结果一致，`make mocks-check` 才稳定。
MOCKERY_VERSION ?= v3.7.1

help:
	@echo "Available targets:"
	@echo "  make run         - Run backend in serve mode"
	@echo "  make dev         - Run backend with Air hot reload"
	@echo "  make air-install - Install Air to GOPATH/bin"
	@echo "  make web-dev     - Run frontend dev server"
	@echo "  make build       - Build local binary with version/commit injected"
	@echo "  make bump NEW_VERSION=X.Y.Z"
	@echo "                   - Bump internal/version.Version + sanity-check (does NOT commit/tag)"
	@echo "  make check       - Backend fmt/lint + web format/lint + i18n checks"
	@echo "  make dev-lint    - Backend fmt/lint + web format/lint + i18n checks"
	@echo "  make lint        - Run golangci-lint checks"
	@echo "  make fmt         - Run golangci-lint formatters"
	@echo "  make test        - Run Go tests"
	@echo "  make test-race   - Run Go tests with the race detector"
	@echo "  make test-cover  - Run Go tests with coverage, print total"
	@echo "  make mocks       - Regenerate testify mocks via mockery (internal/test/mocks)"
	@echo "  make mocks-check - Fail if generated mocks are stale vs code"
	@echo "  make wire        - Generate DI code via Wire"
	@echo "  make wire-check  - Verify Wire code is up-to-date"
	@echo "  make openapi     - Regenerate OpenAPI spec (Huma) to internal/openapi/openapi.yaml"
	@echo "  make openapi-check - Fail if the committed OpenAPI spec is stale"
	@echo "  make spdx        - Add SPDX/Copyright headers to new .go/.ts/.vue files"
	@echo "  make spdx-check  - Fail if any source file is missing the SPDX header"
	@echo "  make build-image - Build Docker image"
	@echo "  make push-image  - Push Docker image"

air-install:
	go install github.com/air-verse/air@latest

run:
	ECH0_SERVER_MODE=debug go run -ldflags "$(LDFLAGS)" ./cmd/ech0 serve

build:
	go build -ldflags "$(LDFLAGS)" -o ./bin/ech0 ./cmd/ech0

# Prepare a clean version-bump commit. This target only EDITS files —
# it never auto-commits, never tags, never pushes. The next-step commands
# are printed for the developer to run after eyeballing the diff.
# See docs/dev/release-process.md for the full procedure.
bump:
	@if [ -z "$(NEW_VERSION)" ]; then \
		echo "✘ Usage: make bump NEW_VERSION=X.Y.Z"; \
		echo "  e.g. make bump NEW_VERSION=4.6.5"; \
		exit 1; \
	fi
	@echo "$(NEW_VERSION)" | grep -Eq '$(SEMVER_PATTERN)' \
		|| { echo "✘ '$(NEW_VERSION)' is not valid semver (expected X.Y.Z[-prerelease])"; exit 1; }
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "✘ Working tree dirty — commit or stash first so the bump commit is clean."; \
		git status --short; \
		exit 1; \
	fi
	@OLD_VERSION="$$(grep -E '^[[:space:]]*Version[[:space:]]*=[[:space:]]*\"' internal/version/version.go \
	                  | head -n1 \
	                  | sed -E 's/.*\"([^\"]+)\".*/\1/')"; \
	if [ -z "$$OLD_VERSION" ]; then \
		echo "✘ Could not extract current Version from internal/version/version.go"; \
		exit 1; \
	fi; \
	if [ "$$OLD_VERSION" = "$(NEW_VERSION)" ]; then \
		echo "✘ Version is already $$OLD_VERSION — nothing to bump."; \
		exit 1; \
	fi; \
	echo "→ bumping $$OLD_VERSION → $(NEW_VERSION)"; \
	sed -i.bak -E 's/^([[:space:]]*Version[[:space:]]*=[[:space:]]*\")[^\"]+(\")/\1$(NEW_VERSION)\2/' internal/version/version.go; \
	rm -f internal/version/version.go.bak
	@echo "→ verifying go build still succeeds..."
	@go build ./... >/dev/null || { echo "✘ go build failed after bump — reverting"; git checkout -- internal/version/version.go; exit 1; }
	@echo ""
	@echo "✓ Version bumped. Diff:"
	@git --no-pager diff -- internal/version/version.go
	@echo ""
	@echo "Next steps (review the diff above, then run):"
	@echo ""
	@echo "  # 1. Update CHANGELOG.md: rename [Unreleased] → [$(NEW_VERSION)] - $$(date -u +%Y-%m-%d), open a new empty [Unreleased]"
	@echo "  # 2. Commit + tag:"
	@echo "       git commit -am 'chore(release): v$(NEW_VERSION)'"
	@echo "       git tag -a v$(NEW_VERSION) -m 'Release v$(NEW_VERSION)'"
	@echo "  # 3. Push to trigger release workflow:"
	@echo "       git push origin main"
	@echo "       git push origin v$(NEW_VERSION)"

dev:
	@if [ ! -x "$(AIR_BIN)" ]; then \
		echo "air not found, installing..."; \
		$(MAKE) air-install; \
	fi
	ECH0_SERVER_MODE=debug "$(AIR_BIN)" -c .air.toml

web-dev:
	cd web && pnpm dev

check:
	$(MAKE) dev-lint

dev-lint:
	@bash scripts/check.sh

lint:
	golangci-lint run

fmt:
	golangci-lint fmt

test:
	go test ./...

# 竞态检测需要 CGO（go-sqlite3 也需要），显式开启避免环境默认值差异。
test-race:
	CGO_ENABLED=1 go test -race ./...

# 覆盖率：原子计数（配合 -race 安全），跑完打印总覆盖率。
# 同时输出 RAW 与 CALIBRATED 两个口径：CALIBRATED 滤掉生成代码（mockery 生成的
# mock 全 0%、Wire 生成的 wire_gen.go 几乎 0%），是衡量「人写代码」覆盖率的诚实口径。
# 仅后处理 profile，不改测试执行 → 确定可复现；CI 的 coverage summary 用同一过滤。
COVER_EXCLUDE := internal/test/mocks/|/wire_gen\.go:
test-cover:
	CGO_ENABLED=1 go test -coverprofile=coverage.out -covermode=atomic ./...
	@grep -v -E '$(COVER_EXCLUDE)' coverage.out > coverage.calibrated.out
	@printf 'RAW        (incl. generated): '; go tool cover -func=coverage.out            | tail -1 | awk '{print $$NF}'
	@printf 'CALIBRATED (excl. generated): '; go tool cover -func=coverage.calibrated.out | tail -1 | awk '{print $$NF}'

# 重新生成 testify mock（输出到 internal/test/mocks/<domain>mock）。mockery 用 go run 固定版本调用，
# 不进 go.mod；生成物含 SPDX 头（scripts/spdx-boilerplate.txt）与「DO NOT EDIT」标记，golangci-lint 自动跳过。
mocks:
	go run github.com/vektra/mockery/v3@$(MOCKERY_VERSION)

# 校验提交的 mock 与当前接口一致（对齐 wire-check / openapi-check 的「生成物入库 + 漂移即失败」习惯）。
mocks-check: mocks
	git diff --exit-code -- internal/test/mocks

wire:
	go generate ./internal/di

wire-check: wire
	git diff --exit-code -- internal/di/wire_gen.go

openapi:
	go run ./cmd/openapi-gen

openapi-check: openapi
	git diff --exit-code -- internal/openapi/openapi.yaml

spdx:
	node scripts/add-spdx-headers.mjs

spdx-check:
	node scripts/add-spdx-headers.mjs --check

build-image:
	@echo "Building image for platform: $(OS)/$(ARCH)"
	docker build --platform $(OS)/$(ARCH) \
		--build-arg TARGETOS=$(OS) \
		--build-arg TARGETARCH=$(ARCH) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(DOCKER_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) -f docker/build.Dockerfile .
push-image:
	docker push $(DOCKER_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

