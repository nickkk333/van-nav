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
# 需要清理的构建产物目录（clean 目标使用）
CLEAN_DIRS := $(DIST_DIR) $(PUBLIC_DIR) $(UI_DIR)/dist

GIT_TAG    := $(shell git describe --tags --always --dirty)
GIT_COMMIT := $(shell git rev-parse --short HEAD)
VERSION    ?= $(if $(GIT_TAG),$(GIT_TAG),dev)
COMMIT     ?= $(if $(GIT_COMMIT),$(GIT_COMMIT),none)
LDFLAGS    := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

IMAGE_NAME ?= mereith/van-nav
IMAGE_TAG  ?= latest
PORT       ?= 6412
# docker-tar 目标导出的镜像平台架构
ARCH       ?= amd64
# 构建 Docker 镜像时使用的 Node 版本，必须与根目录 .nvmrc 一致
NODE_VERSION ?= 24

# 前端包管理器：npm 或 pnpm
NPM        ?= npm

# 跨平台目录操作
# 删除目录：$(call RMDIR,目录1 目录2 ...)
# Windows 下 PowerShell 的 -Path 只接受一个值，多个目录要拆成数组逐个删除，所以这里把空格分隔的目录列表
# 在 PowerShell 里 Split 成数组；目录不存在时不能只靠 -ErrorAction SilentlyContinue
# （PowerShell 仍会返回非 0 退出码让 make 报错），所以用 Test-Path 先判断存在再删除，保证 make clean 可重复执行
# 创建目录：$(MKDIR) 目录
ifeq ($(OS),Windows_NT)
RMDIR = powershell -NoProfile -Command "foreach ($$p in '$(strip $1)'.Split(' ')) { if (Test-Path -Path $$p) { Remove-Item -Recurse -Force -Path $$p } }"
MKDIR = powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path"
LINUX_AMD64_ENV = set "CGO_ENABLED=0" && set "GOOS=linux" && set "GOARCH=amd64" &&
LINUX_ARM64_ENV = set "CGO_ENABLED=0" && set "GOOS=linux" && set "GOARCH=arm64" &&
WINDOWS_AMD64_ENV = set "CGO_ENABLED=0" && set "GOOS=windows" && set "GOARCH=amd64" &&
else
RMDIR = rm -rf $1
MKDIR = mkdir -p
LINUX_AMD64_ENV = CGO_ENABLED=0 GOOS=linux GOARCH=amd64
LINUX_ARM64_ENV = CGO_ENABLED=0 GOOS=linux GOARCH=arm64
WINDOWS_AMD64_ENV = CGO_ENABLED=0 GOOS=windows GOARCH=amd64
endif

# 控制台提示信息：Windows 的 cmd echo 会把引号原样打印出来（Unix 的 sh 不会），而 Unix 下括号要用引号转义，
# 所以按平台分别拼提示命令；文本本身用英文，因为 Windows 控制台默认 GBK 代码页，输出中文会乱码
ifeq ($(OS),Windows_NT)
MSG = echo $(1)
else
MSG = echo '$(1)'
endif

.DEFAULT_GOAL := help

# ------------------------------------------------------------- 帮助
# 注意：Windows 控制台默认是 GBK(936) 代码页，Makefile 是 UTF-8，cmd 直接 echo 中文会乱码
# （例如「已清理构建产物」会显示成「宸叉竻鐞嗘瀯寤轰骇鐗?」），所以面向控制台的输出一律用 ASCII 英文，
# 中文说明只写在注释里
.PHONY: help
help: ## 显示所有可用命令
	@$(call MSG,Van Nav build commands:)
	@$(call MSG,  make install            Install backend and frontend dependencies)
	@$(call MSG,  make ui-dev             Start frontend dev server (port 2333))
	@$(call MSG,  make ui-build           Build frontend into ./public)
	@$(call MSG,  make ui-check           Frontend type check)
	@$(call MSG,  make build              Build local binary bin/$(BINARY))
	@$(call MSG,  make build-linux        Build Linux amd64 static binary (for Docker))
	@$(call MSG,  make build-linux-arm64  Build Linux arm64 static binary)
	@$(call MSG,  make build-windows-amd64 Build Windows amd64 static binary)
	@$(call MSG,  make build-all          Build all release binaries)
	@$(call MSG,  make run                Run locally (default port $(PORT)))
	@$(call MSG,  make dev                Start backend and frontend dev servers together)
	@$(call MSG,  make docker             Build Docker image $(IMAGE_NAME):$(IMAGE_TAG))
	@$(call MSG,  make docker-tar         Export offline-loadable image tar ($(DIST_DIR)/$(BINARY)-docker-$(ARCH).tar))
	@$(call MSG,  make docker-multiarch   Build and push multi-arch image (amd64/arm64))
	@$(call MSG,  make docker-run         Run Docker container)
	@$(call MSG,  make fmt vet test       Format / static check / test)
	@$(call MSG,  make clean              Clean build artifacts)

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

.PHONY: build-windows-amd64
build-windows-amd64: ui-build ## 构建 Windows amd64 静态二进制（发布 Windows 版使用）
	@$(MKDIR) $(DIST_DIR)
	$(WINDOWS_AMD64_ENV) go build -trimpath -tags timetzdata -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY)-windows-amd64.exe .

.PHONY: build-all
build-all: build-linux build-linux-arm64 build-windows-amd64 ## 构建全部平台二进制

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
		--build-arg NODE_VERSION=$(NODE_VERSION) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		-t $(IMAGE_NAME):$(IMAGE_TAG) -t $(IMAGE_NAME):$(VERSION) .

.PHONY: docker-multiarch
docker-multiarch: ## 构建并推送多架构镜像
	docker buildx build --platform linux/amd64,linux/arm64 \
		--build-arg NODE_VERSION=$(NODE_VERSION) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		-t $(IMAGE_NAME):$(IMAGE_TAG) -t $(IMAGE_NAME):$(VERSION) --push .

.PHONY: docker-tar
docker-tar: ## 导出可离线 docker load 的镜像 tar（平台由 ARCH 控制，默认 amd64）
	@$(MKDIR) $(DIST_DIR)
	docker buildx build --platform linux/$(ARCH) --provenance=false --sbom=false \
		--build-arg NODE_VERSION=$(NODE_VERSION) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		-t $(IMAGE_NAME):$(IMAGE_TAG) -t $(IMAGE_NAME):$(VERSION) \
		--output type=docker,dest=$(DIST_DIR)/$(BINARY)-docker-$(ARCH).tar .
	@$(call MSG,Image exported to $(DIST_DIR)/$(BINARY)-docker-$(ARCH).tar)

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
	@$(call RMDIR,$(CLEAN_DIRS))
	@$(call MSG,Cleaned build artifacts: $(CLEAN_DIRS))
