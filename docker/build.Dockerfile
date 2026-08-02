# 本地 / make build-image 用多阶段构建；逻辑对齐：
# - 前端：.github/workflows/release.yml（pnpm + vite → ../template/dist）
# - 后端 go build：同上 workflow 的 STATIC_LDFLAGS 与 ./cmd/ech0/main.go
# - 运行镜像：根目录 Dockerfile 最终阶段（alpine + tzdata、data 目录）

# =================== 前端构建阶段 ===================
FROM node:25-alpine AS frontend-builder

WORKDIR /web

COPY web/package.json web/pnpm-lock.yaml ./

RUN corepack enable

RUN pnpm install --frozen-lockfile

COPY web/ .

# 输出目录见 web/vite.config.ts outDir: ../template/dist → 构建上下文内为 /template/dist
RUN pnpm run build --mode production

# =================== 后端构建阶段 ===================
FROM golang:1.26.4-alpine AS backend-builder

RUN apk add --no-cache git ca-certificates gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /template/dist /app/template/dist

ARG TARGETOS
ARG TARGETARCH
# 构建元数据：commit / build_time。当宿主传入了 ARG（make build-image 默认会传），
# 直接使用；否则回退到容器内 git / date（要求构建上下文包含 .git）。
ARG GIT_COMMIT
ARG BUILD_TIME

# 与 release.yml「Build backend binary with embedded frontend」一致（embed 读取 template/dist）
RUN BUILD="${BUILD_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}" \
    && CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -tags netgo \
    -ldflags="-linkmode external -extldflags '-static' -X github.com/lin-snow/ech0/internal/version.Commit=custom -X github.com/lin-snow/ech0/internal/version.BuildTime=${BUILD}" \
    -o ech0 ./cmd/ech0/main.go

# =================== 最终镜像（对齐根目录 Dockerfile 运行时层）===================
FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache tzdata \
    && mkdir -p /app/data

COPY --from=backend-builder /app/ech0 /app/ech0

RUN chmod +x /app/ech0

EXPOSE 6277

ENTRYPOINT ["/app/ech0"]
CMD ["serve"]
