#!/usr/bin/env bash
# 09-12-setup-dev-environment — 宿主工具链安装脚本（implement.md Phase 1-5）
#
# 需要无沙箱限制执行：本脚本写入 /etc、/usr/local、/root/go、/var/lib/docker。
# 用法：bash setup-env.sh 2>&1 | tee /tmp/setup-env.log
#
# 设计：任一 Gate 失败即 die（不静默换版本、不继续后续阶段）。
# 注意：本脚本开启 pipefail，故**不得**使用 `cmd | head -1` / `cmd | grep -q` 这类
#       会让左侧进程收到 SIGPIPE(141) 的写法；版本断言一律"先捕获变量，再用 case 匹配"。
# 幂等：Phase 1 复用 install-docker.sh（自带版本判断）；Phase 2 校验 sha256 命中则跳过下载；
#       Phase 3/4 apt 与 go install 均可重复执行。

set -uo pipefail

GO_VERSION=1.26.6
GOLANGCI_VERSION=v2.12.2
REPO=/root/LitePan

step() { printf '\n======== %s ========\n' "$*"; }
ok()   { printf '[OK] %s\n' "$*"; }
die()  { printf '\n[FAIL] %s\n' "$*" >&2; exit 1; }
# contains <haystack> <needle> —— 替代 `... | grep -q`，规避 pipefail+SIGPIPE
contains() { case "$1" in *"$2"*) return 0 ;; *) return 1 ;; esac; }

echo "host: $(. /etc/os-release; echo "$PRETTY_NAME") / $(uname -m) / euid=$(id -u)"
echo "date: $(date -Is)"

# ================= Phase 1: Docker =================
step "Phase 1: Docker 29.7.2 + Compose v5.4.0（复用仓库 install-docker.sh）"
cd "$REPO" || die "无法进入 $REPO"
bash install-docker.sh
rc=$?
[ "$rc" -eq 0 ] || die "install-docker.sh EXIT=$rc（见上方输出）"

step "Gate G1: 版本 / daemon / hold"
docker_ver=$(docker --version 2>&1) || die "docker 不可用"
compose_ver=$(docker compose version 2>&1) || die "docker compose 插件不可用"
contains "$docker_ver"  "Docker version 29.7.2" || die "docker 版本 ≠ 29.7.2 -> $docker_ver"
contains "$compose_ver" "v5.4.0"               || die "compose 版本 ≠ v5.4.0 -> $compose_ver"
state=$(systemctl is-active docker)
if [ "$state" != "active" ]; then
  echo "docker 非 active($state)，尝试 systemctl start docker ..."
  systemctl start docker >/dev/null 2>&1
  sleep 3
  state=$(systemctl is-active docker)
fi
[ "$state" = "active" ] || die "docker daemon 非 active -> $state"
holds=$(apt-mark showhold 2>&1)
contains "$holds" "docker-ce" || die "docker-ce 未被 apt-mark hold"
ok "G1 通过：$docker_ver | $compose_ver | daemon=$state"

step "Gate G1b: docker run --rm hello-world（daemon + registry 链路）"
docker run --rm hello-world || die "docker run hello-world 失败"
ok "G1b 通过"

# ================= Phase 2: Go =================
step "Phase 2: Go $GO_VERSION（官方 tarball + sha256 校验）"
TARBALL=/tmp/go${GO_VERSION}.linux-amd64.tar.gz
JSON=/tmp/go-dl.json
[ -f "$JSON" ] || curl -fsSL -o "$JSON" 'https://go.dev/dl/?mode=json&include=all' || die "无法获取 go.dev 发布清单"
EXPECT_SHA=$(python3 - "$JSON" "$GO_VERSION" <<'PY'
import json, sys
d = json.load(open(sys.argv[1]))
ver = sys.argv[2]
want = 'go%s.linux-amd64.tar.gz' % ver
for r in d:
    if r.get('version') == 'go' + ver:
        for f in r.get('files', []):
            if f.get('filename') == want:
                print(f['sha256'])
                raise SystemExit(0)
raise SystemExit(1)
PY
) || die "go$GO_VERSION linux-amd64 不在 go.dev 发布清单中 —— 按 PRD 约束不换版本，停止"
echo "expect sha256: $EXPECT_SHA"

if [ -f "$TARBALL" ] && echo "$EXPECT_SHA  $TARBALL" | sha256sum -c - >/dev/null 2>&1; then
  ok "本地 tarball 已存在且 sha256 吻合，跳过下载"
else
  curl -fsSL -o "$TARBALL" "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" || die "go tarball 下载失败"
  echo "$EXPECT_SHA  $TARBALL" | sha256sum -c - || die "go tarball sha256 校验失败"
fi

if [ -x /usr/local/go/bin/go ] && contains "$(/usr/local/go/bin/go version 2>&1)" "go${GO_VERSION} linux/amd64"; then
  ok "/usr/local/go 已是 go${GO_VERSION}，跳过解包"
else
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "$TARBALL" || die "go 解包失败"
fi
printf 'export PATH=/usr/local/go/bin:$PATH\n' > /etc/profile.d/go.sh
chmod 0644 /etc/profile.d/go.sh
export PATH=/usr/local/go/bin:$PATH

step "Gate G2: go version（当前 shell + 非交互 login shell）"
go_ver=$(go version 2>&1) || die "go 不可用"
contains "$go_ver" "go${GO_VERSION} linux/amd64" || die "go 版本 ≠ ${GO_VERSION} -> $go_ver"
env -i PATH=/usr/bin:/bin HOME=/root bash -lc 'go version' >/dev/null 2>&1 \
  || die "非交互 login shell 下 go 不可用（/etc/profile.d/go.sh 未生效）"
ok "G2 通过：$go_ver"

# ================= Phase 3: make + C toolchain =================
step "Phase 3: make + build-essential（go test -race 的 cgo 前置）"
export DEBIAN_FRONTEND=noninteractive
apt-get install -y make build-essential || die "apt-get install make build-essential 失败"

step "Gate G3: make / gcc / Makefile 可解析"
make_ver=$(make --version 2>&1) || die "make 不可用"
gcc_ver=$(gcc --version 2>&1)  || die "gcc 不可用"
printf '%s\n' "${make_ver%%$'\n'*}" "${gcc_ver%%$'\n'*}"
cd "$REPO" && make -n lint >/dev/null || die "make -n lint 解析失败"
ok "G3 通过：${make_ver%%$'\n'*} | ${gcc_ver%%$'\n'*}"

# ================= Phase 4: golangci-lint =================
step "Phase 4: golangci-lint $GOLANGCI_VERSION"
export PATH=/usr/local/go/bin:$PATH
export GOPROXY=https://goproxy.cn,direct
# 注意：Makefile 钉的是 v2.12.2（带 v），而 golangci-lint 自报版本为 "2.12.2"（不带 v），
# 断言统一用去 v 的数字串。
GL_VER_NUM="${GOLANGCI_VERSION#v}"
BIN="$(go env GOPATH)/bin/golangci-lint"
if [ -x "$BIN" ] && contains "$("$BIN" --version 2>&1)" "$GL_VER_NUM"; then
  ok "golangci-lint 已就绪，跳过安装"
elif ! go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_VERSION}; then
  echo "[WARN] 首次 go install 失败，改用 sumdb 国内镜像重试（不降低校验强度）..."
  GOSUMDB=sum.golang.google.cn go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_VERSION} \
    || die "go install golangci-lint 失败（两次尝试）"
fi

step "Gate G4: golangci-lint 版本"
BIN="$(go env GOPATH)/bin/golangci-lint"
[ -x "$BIN" ] || die "未找到可执行文件 $BIN"
gl_ver=$("$BIN" --version 2>&1) || die "golangci-lint 无法执行"
contains "$gl_ver" "$GL_VER_NUM" || die "golangci-lint 版本 ≠ ${GL_VER_NUM} -> $gl_ver"
ok "G4 通过：$gl_ver"

# ================= Phase 5: 端到端验证 =================
step "Phase 5: 端到端验证（记录结果，不做源码修改）"
cd "$REPO" || die "无法进入 $REPO"
export PATH=/usr/local/go/bin:$PATH
export GOPROXY=https://goproxy.cn,direct

echo "--- 5.1 GOWORK=off go vet ./... ---"
GOWORK=off go vet ./...; echo "[5.1 go vet exit=$?]"

echo "--- 5.2 GOWORK=off go test ./... ---"
GOWORK=off go test ./...; echo "[5.2 go test exit=$?]"

echo "--- 5.2b GOWORK=off go test -race ./internal/store（cgo/-race 链路）---"
GOWORK=off go test -race ./internal/store; echo "[5.2b race exit=$?]"

echo "--- 5.3/5.4 web: npm ci + npm run type-check（刻意不跑 npm run build）---"
cd "$REPO/web" || die "无法进入 $REPO/web"
if ! npm ci --no-audit --no-fund; then
  echo "[WARN] 默认 registry 失败，改用 npmmirror 重试..."
  npm ci --no-audit --no-fund --registry=https://registry.npmmirror.com || die "npm ci 失败（两次尝试）"
fi
npm run type-check; echo "[5.4 type-check exit=$?]"

echo
echo "[DONE] setup-env.sh 执行完毕 $(date -Is)"
