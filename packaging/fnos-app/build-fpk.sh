#!/usr/bin/env bash
# ============================================================
# 打包飞牛OS（fnOS）.fpk 安装包（Docker 模式）
#
# 用法：
#   bash packaging/fnos-app/build-fpk.sh [version] [platform] [image] [output]
#     version  应用版本，写进 manifest（默认 dev）
#     platform 目标平台，x86 或 arm（默认 x86）
#     image    容器内使用的镜像，带 tag 的完整引用（默认 ghcr.io/nickkk333/van-nav:${version}，
#              需要把镜像名或 tag 换成别的时直接整体覆盖）
#     output   产物路径，相对路径基于仓库根目录（默认 bin/van-nav-fnos-amd64.fpk）
#
# 依赖：bash / git / tar / sed / md5sum（Linux、macOS、WSL、Git Bash 均可）
# 说明：生命周期脚本由第三方通用打包框架 conversun/fnos-apps 提供，这里只做
#       应用源组装 + app.tgz + 调用其 build-fpk.sh。
# ============================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
APP_SRC="$SCRIPT_DIR"

VERSION="${1:-dev}"
PLATFORM="${2:-x86}"
IMAGE="${3:-ghcr.io/nickkk333/van-nav:${VERSION}}"
OUTPUT="${4:-bin/van-nav-fnos-amd64.fpk}"

case "$OUTPUT" in
  /*) ;;
  *) OUTPUT="$REPO_ROOT/$OUTPUT" ;;
esac

TOOLKIT_REPO="https://github.com/conversun/fnos-apps"
TOOLKIT_DIR="$(mktemp -d)/fnos-toolkit"
APP_DIR="$TOOLKIT_DIR/apps/van-nav"
trap 'rm -rf "$TOOLKIT_DIR"' EXIT

echo ">>> 获取 fnOS 打包框架"
git clone --depth 1 "$TOOLKIT_REPO" "$TOOLKIT_DIR"

echo ">>> 组装应用目录"
mkdir -p "$APP_DIR/fnos"
cp -a "$APP_SRC"/. "$APP_DIR/fnos/"
rm -f "$APP_DIR/fnos/build-fpk.sh"
# ICON.PNG(128) / ICON_256.PNG(256) 已经在 packaging/fnos-app 下，随上面的 cp -a 一起带过去
sed -i.bak "s|@IMAGE@|${IMAGE}|g" "$APP_DIR/fnos/docker/docker-compose.yaml"
rm -f "$APP_DIR/fnos/docker/docker-compose.yaml.bak"

echo ">>> 生成 app.tgz（Docker 模式只需要 compose + UI）"
( cd "$APP_DIR/fnos" && tar -czf app.tgz docker ui )

echo ">>> 打包 fpk"
mkdir -p "$(dirname "$OUTPUT")"
( cd "$TOOLKIT_DIR" \
  && NAME="$(scripts/build-fpk.sh apps/van-nav apps/van-nav/fnos/app.tgz "$VERSION" "$PLATFORM" | tail -n 1)" \
  && cp "$NAME" "$OUTPUT" )

echo ">>> 完成：$OUTPUT"
ls -lh "$OUTPUT"
