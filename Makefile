# ===================== Van Nav Makefile =====================
# 技术栈：Go(gin + modernc sqlite) + Vue3(Vite + Element Plus)
# 前端产物直接输出到 ./public，后端通过 go:embed 内嵌为单文件可执行程序
#
# 常用命令：
#   make install        安装前后端依赖
#   make build          本地构建可执行文件 bin/van-nav
#   make build-linux    构建 Linux amd64 静态二进制（Docker 镜像使用）
#   make docker         构建 Docker 镜像
#   make dev            同时启动后端与前端开发服务器
# ===========================================================

BINARY     := van-nav
DIST_DIR   := bin
PUBLIC_DIR := public
UI_DIR     := ui

GIT_TAG    := $(shell git describe --tags --always --dirty)
GIT_COMMIT := $(shell git rev-parse --short HEAD)
VERSION    ?= $(if $(GIT_TAG),$(GIT_TAG),dev)
COMMIT     ?= $(if $(GIT_COMMIT),$(GIT_COMMIT),none)
LDFLAGS    := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

IMAGE_NAME ?= mereith/van-nav
IMAGE_TAG  ?= latest
PORT       ?= 6412

# 前端包管理器：npm 或 pnpm
NPM        ?= npm

# 跨平台目录操作
ifeq ($(OS),Windows_NT)
RMDIR = powershell -NoProfile -Command "Remove-Item -Recurse -Force -ErrorAction SilentlyContinue -Path"
MKDIR = powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path"
LINUX_AMD64_ENV = set "CGO_ENABLED=0" && set "GOOS=linux" && set "GOARCH=amd64" &&
LINUX_ARM64_ENV = set "CGO_ENABLED=0" && set "GOOS=linux" && set "GOARCH=arm64" &&
else
RMDIR = rm -rf
MKDIR = mkdir -p
LINUX_AMD64_ENV = CGO_ENABLED=0 GOOS=linux GOARCH=amd64
LINUX_ARM64_ENV = CGO_ENABLED=0 GOOS=linux GOARCH=arm64
endif

.DEFAULT_GOAL := help

# ------------------------------------------------------------- 帮助
.PHONY: help
help: ## 显示所有可用命令
	@echo Van Nav 构建命令:
	@echo "  make install            安装前后端依赖"
	@echo "  make ui-dev             启动前端开发服务器(端口 2333)"
	@echo "  make ui-build           构建前端(输出到 ./public)"
	@echo "  make ui-check           前端类型检查"
	@echo "  make build              本地构建可执行文件 bin/$(BINARY)"
	@echo "  make build-linux        构建 Linux amd64 静态二进制(Docker 使用)"
	@echo "  make build-linux-arm64  构建 Linux arm64 静态二进制"
	@echo "  make run                本地运行(默认端口 $(PORT))"
	@echo "  make dev                同时启动后端与前端开发服务器"
	@echo "  make docker             构建 Docker 镜像 $(IMAGE_NAME):$(IMAGE_TAG)"
	@echo "  make docker-multiarch   构建并推送多架构镜像(amd64/arm64)"
	@echo "  make docker-run         运行 Docker 容器"
	@echo "  make fmt vet test       格式化 / 静态检查 / 测试"
	@echo "  make clean              清理构建产物"

# ------------------------------------------------------------- 依赖
.PHONY: install
install: ui-install ## 安装全部依赖
	go mod download

.PHONY: ui-install
ui-install: ## 安装前端依赖
	cd $(UI_DIR) && $(NPM) install --no-audit --no-fund

.PHONY: tidy
tidy: ## 整理 go 依赖
	go mod tidy

# ------------------------------------------------------------- 前端
.PHONY: ui-build
ui-build: ## 构建前端，产物输出到 ./public
	cd $(UI_DIR) && $(NPM) run build

.PHONY: ui-dev
ui-dev: ## 启动前端开发服务器
	cd $(UI_DIR) && $(NPM) run dev

.PHONY: ui-check
ui-check: ## 前端类型检查
	cd $(UI_DIR) && $(NPM) run type-check

# ------------------------------------------------------------- 构建
.PHONY: build
build: ui-build ## 本地构建可执行文件
	@$(MKDIR) $(DIST_DIR)
	go build -trimpath -tags timetzdata -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY) .

.PHONY: build-linux
build-linux: ui-build ## 构建 Linux amd64 静态二进制（Docker 镜像使用）
	@$(MKDIR) $(DIST_DIR)
	$(LINUX_AMD64_ENV) go build -trimpath -tags timetzdata -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY)-linux-amd64 .

.PHONY: build-linux-arm64
build-linux-arm64: ui-build ## 构建 Linux arm64 静态二进制
	@$(MKDIR) $(DIST_DIR)
	$(LINUX_ARM64_ENV) go build -trimpath -tags timetzdata -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY)-linux-arm64 .

.PHONY: build-all
build-all: build-linux build-linux-arm64 ## 构建全部平台二进制

# ------------------------------------------------------------- 运行
.PHONY: run
run: ## 本地运行
	go run . -port $(PORT)

.PHONY: dev-api
dev-api: ## 启动后端（供前端开发代理）
	go run . -port $(PORT)

.PHONY: dev
dev: ## 同时启动后端与前端开发服务器
	$(MAKE) -j2 dev-api ui-dev

# ------------------------------------------------------------- Docker
.PHONY: docker
docker: ## 构建 Docker 镜像
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		-t $(IMAGE_NAME):$(IMAGE_TAG) -t $(IMAGE_NAME):$(VERSION) .

.PHONY: docker-multiarch
docker-multiarch: ## 构建并推送多架构镜像
	docker buildx build --platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		-t $(IMAGE_NAME):$(IMAGE_TAG) -t $(IMAGE_NAME):$(VERSION) --push .

.PHONY: docker-run
docker-run: ## 运行 Docker 容器
	docker run -d --name van-nav --restart always -p $(PORT):6412 \
		-v "$(CURDIR)/data:/app/data" $(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: docker-stop
docker-stop: ## 停止并删除容器
	-docker rm -f van-nav

# ------------------------------------------------------------- 校验
.PHONY: fmt
fmt: ## 格式化 Go 代码
	go fmt ./...

.PHONY: vet
vet: ## Go 静态检查
	go vet ./...

.PHONY: test
test: ## 运行测试
	go test ./...

.PHONY: check
check: fmt vet ui-check ## 完整检查

.PHONY: clean
clean: ## 清理构建产物
	@$(RMDIR) $(DIST_DIR) $(PUBLIC_DIR) $(UI_DIR)/dist
	@echo 已清理构建产物
