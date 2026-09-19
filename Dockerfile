# syntax=docker/dockerfile:1

# ============================================================
# Van Nav 多阶段构建
#   1. Node 构建 Vue3 前端（产物输出到 /app/public）
#   2. Go 编译静态二进制（go:embed 内嵌前端产物 + 内嵌时区数据）
#   3. alpine 运行，最终镜像只包含一个可执行文件
# ============================================================

ARG GO_VERSION=1.26
ARG NODE_VERSION=24

# ------------------------------------------------------------
# 阶段一：构建前端（Vue 3 + Vite + Element Plus）
# ------------------------------------------------------------
FROM node:${NODE_VERSION}-alpine AS frontend-builder

WORKDIR /app/ui
COPY ui/package.json ui/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --no-audit --no-fund
COPY ui/ ./
# vite.config.ts 中 outDir 为 ../public
RUN npm run build

# ------------------------------------------------------------
# 阶段二：构建后端（Go 静态二进制）
# 说明：modernc.org/sqlite 为纯 Go 实现，CGO_ENABLED=0 即可静态编译
#      -tags timetzdata 会把时区数据打进二进制，运行镜像无需 tzdata
# ------------------------------------------------------------
FROM golang:${GO_VERSION}-alpine AS backend-builder

ARG GOPROXY=https://proxy.golang.org,direct
ARG VERSION=dev
ARG COMMIT=none
ENV GOPROXY=${GOPROXY}
ENV CGO_ENABLED=0

WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
# 使用前端构建产物作为 go:embed 的 public 目录
COPY --from=frontend-builder /app/public ./public
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -tags timetzdata \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o /app/van-nav .

# ------------------------------------------------------------
# 阶段三：运行镜像（不执行 apk，避免构建时依赖软件源网络）
# ------------------------------------------------------------
FROM alpine:3.22

ENV TZ=Asia/Shanghai

# 根证书从构建阶段复制，用于抓取网站图标等 HTTPS 请求
COPY --from=backend-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

WORKDIR /app
COPY --from=backend-builder /app/van-nav /app/van-nav

# 数据库文件默认生成在 /app/data/nav.db
VOLUME ["/app/data"]
EXPOSE 6412

ENTRYPOINT ["/app/van-nav"]