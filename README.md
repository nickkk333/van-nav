# Van Nav

一个轻量的导航站，现在有搜索引擎集成了，很适合作为主页使用。有配套的[浏览器插件](https://github.com/Mereithhh/van-nav-extension)和 API。 [在线体验](https://demo-tools.mereith.com) (总有人改后台数据，后台密码就不放出来了)

> 新增了 [API 文档](https://van-nav-api.mereith.dev)，用 AI 生成的，如果不准确请提 Issue 哦。

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
make docker                       # 等价于 docker build -t mereith/van-nav:latest .

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
- 搜索完按回车会直接在新标签页打开第一个结果。
- 搜索完按一下对应卡片右上角的数字按钮 + Ctrl(mac 也可以用 command 键) ，也会直接打开对应结果。

另外可以设置跳转方式哦。

后台「系统设置」里可以直接上传图片：首页背景图、网站 logo、logo192 / logo512 都支持上传（也可以直接填写图片外链）；上传的图片保存在 `data/images` 下。首页背景图留空时会自动使用**必应每日壁纸**：服务端启动时下载当天的壁纸保存到 `data/必应壁纸.jpg`，前台通过 `/api/bingWallpaper` 读取（每天只下载一次，离线或下载失败时沿用上一次保存的图片，不影响启动）。

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
docker run -d --name tools --restart always -p 6412:6412 -v /path/to/your/data:/app/data mereith/van-nav:latest
```

打开浏览器 [http://localhost:6412](http://localhost:6412) 即可访问。

也可以参照 `Makefile` 中的 `docker` / `docker-run` 命令：

```bash
make docker       # 构建镜像
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

数据落在 fnOS 的**共享目录** `/vol1/@appshare/Van Nav`：由 `packaging/fnos-app/config/resource` 的 `data-share` 声明，安装时读 `TRIM_DATA_SHARE_PATHS` 写进 compose，挂载到容器的 `/app/data`。在飞牛的文件管理器里能直接看到 `nav.db`、`images/`、图片缓存和启动自动备份。

> 实现见 `packaging/fnos-app/docker/docker-compose.yaml` 与 `cmd/service-setup`：compose 用 `${TRIM_DATA_SHARE_PATHS:-默认目录}` 取共享目录（compose 不支持 bash 的 `${VAR%%:*}` 截取），安装/升级钩子再把默认值改写成真实共享路径，保证变量没注入进 compose 环境时也不会挂错目录。
> 旧版用的是 `@appdata/van-nav/data`，要迁移的话把里面的 `nav.db` 和 `images/` 拷到共享目录 `Van Nav` 即可。

> fpk 内跑的是 GHCR 在线镜像（`ghcr.io/<owner>/<repo>:<版本>`），**需要先在 GitHub 包设置里把该镜像设为 Public**，否则飞牛拉取镜像会失败。
> 本地也可以打包：`make fnos-fpk`（需要 bash 环境，产物在 `bin/van-nav-fnos-amd64.fpk`）。

安装时报 `Get "https://registry-1.docker.io/v2/": context deadline exceeded` 是**镜像拉取网络问题**，按下面任一方式处理：

1. 飞牛 → Docker → 设置里配置镜像加速（registry mirror），让 Docker 能访问镜像仓库。
2. 离线安装：先在飞牛上导入离线镜像，再装 fpk（fpk 内 compose 引用的镜像名要与导入的一致）。
   ```bash
   gunzip -c van-nav-docker-amd64.tar.gz | docker load
   ```
3. 用自建/国内仓库地址重新打包 fpk（镜像仓库与标签都可覆盖）：
   ```bash
   make fnos-fpk FNOS_IMAGE=docker.m.daocloud.io/ghcr.io/nickkk333/van-nav FNOS_IMAGE_TAG=v1.0.0
   ```

### nginx 反向代理

参考配置

> 其中 `<yourhost>` 和 `<your-cert-path>` 替换成你自己的。

```
server {
    listen 80;
    server_name <yourhost>;
    return 301 https://$host$request_uri;
}

server {
    listen 443   ssl http2;
    server_name <yourhost>;

    ssl_certificate <your-cert-path>
    ssl_certificate_key <your-key-path>;
    ssl_verify_client off;
    proxy_ssl_verify off;
    location / {
        proxy_pass  http://127.0.0.1:6412;
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_redirect off;
        proxy_set_header Upgrade $http_upgrade;
    }
}
```

### systemd 服务

可以注册成系统服务，开机启动。

1. 复制二进制文件到 `/usr/local/bin` 目录下，并加上执行权限

2. 新建 `VanNav.serivce` 文件于 `/usr/lib/systemd/system` 目录下:

```
[Unit]
Description=VanNav
Documentation=https://github.com/mereithhh/van-nav
After=network.target
Wants=network.target

[Service]
WorkingDirectory=/usr/local/bin
ExecStart=/usr/local/bin/nav
Restart=on-abnormal
RestartSec=5s
KillMode=mixed

StandardOutput=null
StandardError=syslog

[Install]
WantedBy=multi-user.target
```

3. 执行:

```
sudo systemctl daemon-reload && sudo systemctl enable --now VanNav.service
```

## 浏览器插件

具体请看： [浏览器插件仓库](https://github.com/Mereithhh/van-nav-extension)

具有一键增加工具，快速打开管理后台和主站等功能。具体自行探索哦。

## API

本导航站支持 API，可以用自己的方法添加工具。

尝试用 ai 生成 api 文档，具体请看

> [API 文档](https://van-nav-api.mereith.dev)

## FAQ

- 忘记密码了怎么办： [看这里](https://github.com/Mereithhh/van-nav/issues/36)

## 参与开发

前端已重构为 **Vue 3 + Element Plus**（`ui/` 目录），后端为 **Go + gin + sqlite**，两者的接口约定见 `ui/src/api/index.ts` 与 `handler/handlers.go`。

如果你有 golang 和 vue3 开发经验，可以很轻松上手。修改前端时使用 `make ui-dev` 启动开发服务器即可，接口会自动代理到本地后端。

如果没有方向，可以试试去解决 issue 里的问题或者开发新功能，开发之前可以先提个 issue 让我知道。

## 状态

可以优化的点太多了，慢慢完善吧……

- [x] 多平台构建流水线
- [x] 定制化 logo 和标题
- [x] 导入导出功能
- [x] 暗色主题切换
- [x] 移动端优化
- [x] 自动获取网站 logo
- [x] 拼音匹配的模糊搜索功能
- [x] 按键直接搜索，搜索后回车直接打开第一项
- [x] 图片存库，避免跨域和加载慢的问题
- [x] gzip 全局压缩
- [x] 中文 url 图片修复
- [x] svg 图片修复
- [x] 浏览器插件
- [x] 自动获取网站题目和描述等信息
- [x] 后台按钮可自定义隐藏
- [x] github 按钮可隐藏
- [x] 支持登录后才能查看的隐藏卡片
- [x] 搜索引擎集成功能
- [x] 增加一些搜索后快捷键直接打开卡片
- [x] 支持自定义跳转方式
- [x] 自动主题切换
- [ ] 国际化
- [x] 增加 ServiceWork ,离线可用,可安装
- [ ] 网站状态检测
- [x] 支持后台设置默认跳转方式
- [x] 支持指定监听端口
