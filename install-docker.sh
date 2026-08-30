#!/bin/bash
# 一键安装项目标准 Docker 环境：Docker 29.7.2 + Compose v5.4.0（Debian/Ubuntu）- 修复版
# 用法：curl -fsSL http://<LitePan-IP>:5211/install-docker.sh | bash
#    或：bash install-docker-fixed.sh
# 修复点：1) 已装 docker 但版本非 29.7.2 时仍重建官方源 2) 显式 signed-by 建源 3) rootless-extras 缺失容错
set -euo pipefail

REQUIRED_DOCKER="29.7.2"
REQUIRED_COMPOSE="v5.4.0"

if [ "$(id -u)" -ne 0 ]; then
  echo "请用 root 运行（sudo bash install-docker-fixed.sh）" >&2
  exit 1
fi

if ! grep -qiE 'debian|ubuntu' /etc/os-release 2>/dev/null; then
  echo "仅支持 Debian/Ubuntu，其他系统请手动安装 Docker $REQUIRED_DOCKER + Compose $REQUIRED_COMPOSE" >&2
  exit 1
fi

echo "==> 安装 Docker $REQUIRED_DOCKER + Compose $REQUIRED_COMPOSE (修复版)"

# 检测当前版本，决定是否强制重建官方源
NEED_REINSTALL=false
if command -v docker >/dev/null 2>&1; then
  CUR_DOCKER=$(docker --version 2>/dev/null | grep -oP "Docker version \K[0-9.]+" || echo "missing")
  if [ "$CUR_DOCKER" != "$REQUIRED_DOCKER" ]; then
    echo "检测到 docker $CUR_DOCKER ≠ $REQUIRED_DOCKER，将重建官方源..."
    NEED_REINSTALL=true
  else
    echo "检测到 docker $CUR_DOCKER 已为目标版本，仍校验官方源..."
  fi
else
  NEED_REINSTALL=true
fi

# 显式建官方源（替代 get.docker.com 的隐式，确保 bookworm 等 signed-by 路径明确）
if [ "$NEED_REINSTALL" = true ] || [ ! -f /etc/apt/sources.list.d/docker.list ]; then
  echo "配置 Docker 官方源..."
  mkdir -p /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg
  . /etc/os-release
  # Debian 与 Ubuntu 共用 debian 源亦可，官方推荐按 ID 区分
  REPO_URL="https://download.docker.com/linux/debian"
  if grep -qi 'ubuntu' /etc/os-release; then
    REPO_URL="https://download.docker.com/linux/ubuntu"
  fi
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] $REPO_URL $VERSION_CODENAME stable" > /etc/apt/sources.list.d/docker.list
fi

# 若 docker 未装，仍走 get.docker.com 兜底（会再次建源，无害）
if ! command -v docker >/dev/null 2>&1; then
  echo "未检测到 docker，执行 get.docker.com 兜底..."
  curl -fsSL https://get.docker.com | sh
fi

export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
# rootless-extras 在部分 Ubuntu 的 docker.io 已装场景下 has no candidate，做容错
apt-get install -y --allow-downgrades \
  docker-ce=5:${REQUIRED_DOCKER}* \
  docker-ce-cli=5:${REQUIRED_DOCKER}* \
  docker-compose-plugin=${REQUIRED_COMPOSE#v}* \
  docker-buildx-plugin containerd.io || {
  echo "首次安装失败，尝试容错安装（跳过 rootless-extras）..." >&2
  apt-get install -y --allow-downgrades \
    docker-ce=5:${REQUIRED_DOCKER}* \
    docker-ce-cli=5:${REQUIRED_DOCKER}* \
    docker-compose-plugin=${REQUIRED_COMPOSE#v}* \
    docker-buildx-plugin containerd.io
}
# rootless-extras 单独容错
apt-get install -y docker-ce-rootless-extras 2>&1 | tail -5 || echo "docker-ce-rootless-extras 跳过（Ubuntu 已装 docker.io 场景常见）" >&2

apt-mark hold docker-ce docker-ce-cli docker-ce-rootless-extras docker-buildx-plugin docker-compose-plugin containerd.io >/dev/null 2>&1 || true

systemctl enable --now docker >/dev/null 2>&1 || service docker start >/dev/null 2>&1 || true

DOCKER_VER=$(docker --version 2>/dev/null | grep -oP "Docker version \K[0-9.]+" || echo "missing")
COMPOSE_VER=$(docker compose version 2>/dev/null | grep -oP "Docker Compose version \K\S+" || echo "missing")

if [ "$DOCKER_VER" = "$REQUIRED_DOCKER" ] && [ "$COMPOSE_VER" = "$REQUIRED_COMPOSE" ]; then
  echo "✅ 完成：Docker $DOCKER_VER + Compose $COMPOSE_VER 已就绪（已 hold 锁定版本）"
else
  echo "⚠️ 安装后版本仍不符：Docker $DOCKER_VER（期望 $REQUIRED_DOCKER），Compose $COMPOSE_VER（期望 $REQUIRED_COMPOSE）" >&2
  echo "请检查 apt 源或手动执行：apt-get install -y docker-ce=5:${REQUIRED_DOCKER}* docker-ce-cli=5:${REQUIRED_DOCKER}* docker-compose-plugin=${REQUIRED_COMPOSE#v}*" >&2
  exit 1
fi
