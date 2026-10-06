# Van Nav

一个轻量的导航站，现在有搜索引擎集成了，很适合作为主页使用。有配套的[浏览器插件](https://github.com/nickkk333/van-nav-extension)。

> 本项目 fork 自 [Mereithhh/van-nav](https://github.com/Mereithhh/van-nav)，并经过重构。

## 技术栈

重构后使用的前后端分离 + 单文件打包方案：

| 层 | 技术 |
| --- | --- |
| 后端 | Go 1.26、gin v1.12、gin-contrib/gzip、golang-jwt/v5、modernc.org/sqlite（纯 Go，无需 CGO） |
| 前端 | Vue 3.5、Vite 8、TypeScript 5.9、Element Plus 2.14、Pinia 4、Vue Router 5、axios、pinyin-pro |
| 打包 | 前端构建产物输出到 `public/`，由 Go `//go:embed` 内嵌，最终产出**单个可执行文件** |

## 目录结构

```
.
├── ui/                  # 前端（Vue3 + Vite + Element Plus）
│   ├── src/
│   │   ├── api/         # 后端接口封装
│   │   ├── components/  # 首页组件（搜索框、标签、卡片等）
│   │   ├── composables/ # 组合式函数（表格拖拽排序）
│   │   ├── router/      # 路由
│   │   ├── stores/      # Pinia 状态
│   │   ├── styles/      # 全局样式与暗色主题变量
│   │   ├── types/       # 类型定义
│   │   ├── utils/       # 主题、跳转方式、拼音搜索等工具
│   │   └── views/       # 页面（首页、登录、后台各标签页）
│   ├── public/          # 静态资源（图标等，构建时拷贝到 ./public）
│   └── vite.config.ts   # 构建输出到 ../public
├── public/              # go:embed 目录（由前端构建生成，已 gitignore）
├── database/            # sqlite 初始化、迁移与查询
├── handler/             # gin 路由处理函数
├── service/             # 业务逻辑
├── middleware/          # JWT 鉴权中间件
├── utils/ logger/ types/ goscraper/
├── Makefile             # 常用构建命令
└── Dockerfile           # 多阶段构建（前端 -> 后端 -> 运行镜像）
```

## 本地开发

> 前置条件：Go 1.26 + Node 24。Node 版本统一在根目录 `.nvmrc` 中定义（当前 `24`），`Dockerfile` 的 `NODE_VERSION` 与 CI 流水线都以此为准，改版本只需改这一处。

```bash
# 安装依赖
make install          # 等价于 npm --prefix ui install + go mod download

# 方式一：同时启动后端(6412)和前端开发服务器(2333)
make dev

# 方式二：分别启动
make dev-api          # 后端，监听 6412
make ui-dev           # 前端，监听 2333 并代理 /api 到 6412
```

浏览器访问 <http://localhost:2333> 即可（开发模式）。

常用命令：

```bash
make build            # 构建前端 + 编译本机可执行文件 bin/van-nav
make build-linux      # 构建 Linux amd64 静态二进制（Docker 镜像使用）
make build-linux-arm64
make build-windows-amd64
make build-all        # Linux amd64/arm64 + Windows amd64
make docker-tar       # 导出可 docker load 的离线镜像包（默认 amd64，可用 ARCH=arm64 指定）
make ui-check         # 前端 vue-tsc 类型检查
make fmt vet test     # Go 格式化 / 静态检查 / 测试
make clean            # 清理产物
```

## 构建镜像

```bash
# 构建本地 Docker 镜像
make docker                       # 构建本 fork 镜像（默认标签 ghcr.io/nickkk333/van-nav:latest）

# 导出可离线 docker load 的镜像 tar
make docker-tar

# 多架构构建并推送
make docker-multiarch

# 直接使用 Dockerfile（三阶段：Node 构建前端 -> Go 编译 -> alpine 运行）
docker build --build-arg GOPROXY=https://goproxy.cn,direct -t van-nav:latest .
```

> 国内网络构建时可传 `--build-arg GOPROXY=https://goproxy.cn,direct` 加速 Go 依赖下载。
> 镜像构建时会自动注入版本号（`--build-arg VERSION=... --build-arg COMMIT=...`），`make docker` 已从 git 自动获取。

## 预览

### PC

<img src="images/pc_preview.png" alt="PC" style="width: 100%;"/>

### PAD

<img src="images/pad_preview.png" alt="PAD" style="width: 100%;"/>

### PHONE

<img src="images/phone_preview.png" alt="PHONE" style="width: 100%;"/>

### 后台设置

<img src="images/login.jpg" alt="登录" style="width: 100%;"/>

<img src="images/admin.jpg" alt="后台设置" style="width: 100%;"/>

### 交流群

<img src="images/qqqun.jpg" alt="交流群" style="height: 200px;"/>

> qq 交流群： 873773083

## 使用技巧/快捷键

其实这个导航站有很多小设计，合理使用可以提高使用效率：

- 只要在这个页面里，直接输入键盘任何按键，可以直接聚焦到搜索框开始输入。
- 按回车优先打开**第一个匹配的工具卡片**；没有任何工具卡片匹配时，才用**默认搜索引擎**搜索（搜索引擎在后台「系统设置 → 默认搜索引擎」里配置）。
- 搜索框支持**域名补全**：输入域名前缀（如 `git`）时框内灰显剩余部分（如 `hub.com`），按 `Tab` 自动补全。
- 输入像域名（含 `.com`/`.cn` 等、至少两段、无空格和协议符号）时，按回车或 `Ctrl + Enter` 会**直接访问该域名**，而不是搜索。
- `Ctrl + Enter`：忽略卡片匹配，直接用**默认搜索引擎**搜索（不考虑是否启用），结果在新标签页打开、保留导航页。
- 默认搜索引擎设为特定引擎时：回车有启用的引擎就用第一个启用的，全禁用时才用选中的那个；`Ctrl + Enter` 始终用选中的那个。
- 搜索完按一下对应卡片右上角的数字按钮 + `Ctrl`/`⌘` ，也会直接打开对应结果。

另外可以设置跳转方式哦。

后台「系统设置」里可以直接上传图片：首页背景图、网站 logo、logo192 / logo512 都支持上传（也可以直接填写图片外链）；上传的图片保存在 `data/images` 下。首页背景图留空时会自动使用**必应每日壁纸**，三种触发方式（互相兜底）：

| 时机 | 说明 |
| --- | --- |
| 开机/启动 | 进程启动时异步下载一次 |
| 每天固定时间点 | 默认每天 08:00（本地时区）定时触发 |
| 每天首次访问 | 兜底：本地一张壁纸都没有时才同步拉一次（已有图片不重复下载） |

壁纸保存为 `data/必应壁纸-YYYYMMDD-图片描述.后缀`，**历史壁纸只增不删**；离线或下载失败时沿用上一次保存的图片，不影响启动。

## CHANGELOG

具体请看 [CHANGELOG.md](CHANGELOG.md)

## 发布产物（GitHub Release）

推送 `v*` tag 后会自动发布，Release 中包含：

| 产物 | 说明 |
| --- | --- |
| `van-nav_<版本>_windows_amd64.zip` | Windows amd64 单文件可执行程序 |
| `van-nav_<版本>_linux_amd64.tar.gz` | Linux amd64 静态二进制（另有 arm64 / arm 版本） |
| `van-nav-docker-amd64.tar.gz` | Docker 离线镜像包，`gunzip -c ... \| docker load` 后即可运行（另有 arm64 版） |
| `van-nav-fnos-amd64.fpk` | 飞牛OS（FnOS）amd64 安装包，可在应用中心「手动安装」导入（Docker 模式） |
| `checksums.txt` | 上述二进制包的校验值 |

同时会把多架构镜像推送到 GHCR：`ghcr.io/<owner>/<repo>:latest` 与 `ghcr.io/<owner>/<repo>:<tag>`。

> 打 tag 之前建议先确认 `Check` 流水线（`check.yml`）为绿：它会执行 `goreleaser check`、用 `.nvmrc` 的 Node 构建前端，并在非 PR 场景跑一次不发布的快照构建。
> 本地等价命令：`make goreleaser-check`（快速校验配置）、`make goreleaser-snapshot`（完整演练，产物落在 `dist/`）。

## 安装方法

### Docker

```
docker run -d --name van-nav --restart always -p 6412:6412 -v /path/to/your/data:/app/data ghcr.io/nickkk333/van-nav:latest
```

打开浏览器 [http://localhost:6412](http://localhost:6412) 即可访问。

也可以参照 `Makefile` 中的 `docker` / `docker-run` 命令：

```bash
make docker       # 构建本 fork 镜像（默认 ghcr.io/nickkk333/van-nav:latest）
make docker-run   # 启动容器（映射 6412，挂载 ./data）
```

环境变量：

| 变量 | 说明 | 默认 |
| --- | --- | --- |
| `NAV_JWT_SECRET` | JWT 签名密钥，不设置时每次启动随机生成（重启后旧登录态失效） | 随机 |
| `-port` / `-addr` | 监听端口与地址（启动参数，非环境变量） | 6412 / 0.0.0.0 |

- 默认端口 6412
- 默认账号密码 admin admin 第一次运行后请进入后台修改
- 数据库会自动创建在当前文件夹中： `nav.db`

### 可执行文件

下载 release 文件夹里面对应平台的二进制文件，直接运行即可。

打开浏览器 [http://localhost:6412](http://localhost:6412) 即可访问。

- 默认端口 6412 动时添加 `-port <port>` 参数可指定运行端口。
- 默认监听地址 `0.0.0.0`，可添加 `-addr <addr>` 参数指定。
- 默认账号密码 admin admin ，第一次运行后请进入后台修改
- 数据库会自动创建在当前文件夹中： `nav.db`

也可以自行编译带前端界面的单文件程序：

```bash
make build            # 本机平台，产物在 bin/van-nav
make build-linux      # Linux amd64 静态二进制
```

### 飞牛OS（FnOS）

1. 在 Release 页面下载 `van-nav-fnos-amd64.fpk`。
2. 飞牛桌面 → **应用中心** → 左下角 **手动安装** → 上传该 `.fpk` → 确定。
3. 安装完成后打开桌面图标，或访问 `http://<飞牛IP>:6412`，默认账号密码 `admin` / `admin`。


## 浏览器插件

具体请看： [浏览器插件仓库](https://github.com/nickkk333/van-nav-extension)

具有一键增加工具，快速打开管理后台和主站等功能。具体自行探索哦。

## 参与开发

前端已重构为 **Vue 3 + Element Plus**（`ui/` 目录），后端为 **Go + gin + sqlite**，两者的接口约定见 `ui/src/api/index.ts` 与 `handler/handlers.go`。

如果你有 golang 和 vue3 开发经验，可以很轻松上手。修改前端时使用 `make ui-dev` 启动开发服务器即可，接口会自动代理到本地后端。
