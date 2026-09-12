# 部署 LitePan v0.0.44 到本机 :5211（docker compose，含数据持久化验证）

## Goal

把已发布的 `v0.0.44` 以容器形式部署到本机 `:5211`，用仓库 tracked 的 `docker-compose.yml`（**不改该文件**），数据落在 `/root/LitePan/data` 并验证**可持久化**。

首次启动会创建全新库，管理员为默认口令且强制改密 —— **用户明确表示自行改密，本任务不改口令**。

## Background（2026-09-12 只读侦察）

| 事实 | 值 |
|---|---|
| Docker / Compose | `29.7.2` + `v5.4.0`，daemon `active` |
| 目标镜像 | `ghcr.io/zhemed/litepan:v0.0.44` 本地已在（161MB，digest `sha256:a864057d…`） |
| 端口 | `5211`/`42069` 空闲；`3080`/`3081` 由 node 进程持有（DSH，不可动） |
| 磁盘 | `/` 230G 可用 |
| 运行时目录 | `/root/LitePan/{data,mounts}` **尚不存在**；`.gitignore` 已含 `/data/`、`/mounts/` |
| FUSE | `/dev/fuse` 存在、`fuse3` 已装（容器内镜像亦含 fuse3） |
| 两个 compose 变体 | 根 `docker-compose.yml`：`ports: 5211:5211` + 相对路径 `./data`、`./mounts`，**本机适用**；`docker-compose.fnos.yml`：`network_mode: host` + `/vol1/1000/...` 绝对路径，**飞牛 NAS 专用** |

选根 `docker-compose.yml` 的依据：相对路径自包含、不硬编码 NAS 路径、`ports` 比 host 网络更收敛（只暴露 5211，不带出 42069）。`docker-compose.fnos.yml` 与 README 快速开始同源（均为 NAS 用），本机不使用。

## Requirements

### D1 起容器（用 tracked compose，不修改它）

- 在 `/root/LitePan` 执行 `docker compose up -d`
- 容器名 `litepan`，镜像 `ghcr.io/zhemed/litepan:v0.0.44`
- 挂载 `./data:/app/data`、`./mounts:/app/mounts:shared`、`/dev/fuse`，`privileged: true`、`pid: "host"`、`TZ=Asia/Shanghai`（全部由 compose 提供）
- 若已存在同名容器：先 `docker compose down`（**保留** `data/`）再 up

### D2 可用性验证

- `GET /api/health` → 200 且 `status=ok`
- `GET /` → SPA（含 `LitePan` 的 index.html）
- `POST /api/auth/login`（**表单编码**）用默认口令 → 200 且 `is_admin:true`
- 带 session 取 `GET /api/public/system-config` → `version = v0.0.44`（**顺带验证上一任务的成果在部署形态下成立**）

### D3 数据持久化验证（本任务的核心，不止"能打开"）

- 宿主机 `/root/LitePan/data/litepan.db` 存在
- 容器内 `/app/data/litepan.db` 与宿主机为**同一 bind mount**（比对 inode 或 size+mtime）
- `docker compose restart` 后：`health` 仍 200、且**数据未被重置**（验证方式：重启前记录 DB 大小/登录 session，重启后复测）
- `docker compose ps` 显示 `restart` 策略为 `unless-stopped`

### D4 首次启动状态如实说明（不改口令）

- 本机是**全新库**：管理员为默认口令，响应含 `must_change_password:true`、`password_change_reason:default_credentials`
- **本任务不修改管理员口令**，只在交付说明中告知用户入口（`http://<本机IP>:5211`）与改密路径
- 记录"本机默认口令 ≠ `AGENTS.md` 所记 `123456`"（后者是上一台机器经授权重置后的值），避免后续会话混淆

### D5 边界与不受影响项

- DSH `3080`/`3081` 与其 node 进程**不受任何影响**
- 不改 `docker-compose.yml`、不改任何源码、不改 `AGENTS.md`
- `data/`、`mounts/` 属 gitignored，**不得入库**

## Constraints

- **不改仓库受跟踪文件**：本任务只做宿主/容器操作；`git status` 不得出现受跟踪文件改动（`data/`、`mounts/` 已被 gitignore）
- **不改管理员口令**（用户自理）
- **不占用 `3080`/`3081`**；只使用项目既定端口 `5211`
- **失败不得留半启动状态**：若 up 失败或验证不过，执行 `docker compose down`（保留 `data/`）后再报告，不留一个"看起来在跑但不可用"的容器
- **不得删除 `data/`**：回滚只 down 容器，数据目录保留
- 登录必须用**表单编码**（发 JSON 会被静默解析为空用户名，AGENTS.md 明载）
- 验证结束后 `:5211` 应处于**正常运行**状态（这是交付物，不是临时验证容器）

## Acceptance Criteria

- [x] `docker compose up -d` 成功，`docker compose ps` 显示 `litepan` 为 `Up` —— 实测 `Network litepan_default Created` → `Container litepan Started`；`docker compose ps` = `litepan | ghcr.io/zhemed/litepan:v0.0.44 | Up`
- [x] `ss -ltnp | grep 5211` 显示 `5211` 正在监听 —— 实测 `0.0.0.0:5211` 与 `[::]:5211` 各一条（docker-proxy）
- [x] `curl /api/health` → HTTP 200 且含 `"status":"ok"` —— 实测 `{"success":true,"data":{"boot_id":"70618a3b-…","status":"ok"},…}`
- [x] `curl /` → SPA `index.html` 且含 `LitePan` —— 实测返回 `<!DOCTYPE html><html lang="zh-CN">…`，含 LitePan 标识
- [x] 表单编码登录（默认口令）→ 200 且 `is_admin:true`、`must_change_password:true` —— 实测 `{"username":"admin","is_admin":true,"must_change_password":true,"password_change_reason":"default_credentials"}`
- [x] 带 session 取 `system-config` → `version = v0.0.44`，既有 3 字段均在 —— 实测 `version='v0.0.44'`，三字段 `compact_home_enabled`/`header_effects_enabled`/`index_account_switch_mode` 齐全
- [x] 宿主机 `data/litepan.db` 存在；与容器内 `/app/data/litepan.db` 为同一 bind mount —— 实测两侧 `stat` 均为 `inode=13533519 size=4096`，**逐字节一致**
- [x] `docker compose restart` 后 `health` 仍 200 且**数据未重置** —— 实测重启后首次探测即 200；四条独立证据：① 同一 inode `13533519` 全程未变；② 直接只读读库得 11 张表 + `admin_password = pbkdf2:sha256:600000$151f73a…`；③ `secret.key` inode 未变、mtime 仍是创建时刻；④ **重启前的 session cookie 仍通过鉴权**（无 session 表 → 会话由 `secret.key` 签名，其存活即证明密钥持久化）
- [x] `docker logs litepan` 含 `HTTP 服务已监听` —— 实测 `msg="HTTP 服务已监听" module=system addr=:5211`
- [x] restart 策略为 `unless-stopped` —— 实测 `docker inspect litepan → RestartPolicy = unless-stopped`
- [x] DSH 未受影响：`3080`/`3081` 仍由原 node 进程监听 —— 实测 PID **与基线完全一致**（3080=node/39271、3081=node/9673），未被惊动
- [x] `git status --short` 无受跟踪文件改动 —— 实测仅 `?? .trellis/tasks/09-12-deploy-litepan-local-5211/`；`git check-ignore` 证实 `.gitignore:2 /data/` 与 `:4 /mounts/` 生效
- [x] 交付说明含入口 URL、默认口令处置提示、回滚方式 —— 见下方「交付说明」

## Notes

- Scope 标注 `infra`（宿主/容器基础设施操作 + 数据持久化契约 + 回滚方案），故按复杂任务补齐 `design.md` + `implement.md` 后再 `start`。
- 本任务**不**做：修改 compose、加 healthcheck、改端口、配置反代/TLS、创建网盘账号、改口令。
- **本任务不解决**（如实记录，属另一议题）：`docker-compose.fnos.yml` 的镜像 tag 仍停在 `v0.0.31`（落后 13 个版本）。它是 NAS 专用文件，本机不加载，且把它 bump 到 v0.0.44 会把 13 个版本的变更一次性推给 NAS 用户、而本机无法验证 fnOS 环境 —— 需用户单独决定。
- 用户已明确指示"部署到本机吧，我去修改默认密码"，故本任务不含改密步骤。

## 交付说明

| 项 | 内容 |
|---|---|
| **访问入口** | 本机 `http://127.0.0.1:5211` · 局域网 `http://10.0.0.91:5211`（`0.0.0.0:5211` 已发布） |
| **管理员** | 全新库默认口令，登录后 `must_change_password:true` → **请立即在界面改密**（本任务未改口令，入口：登录后右上角账户菜单） |
| **数据位置** | `/root/LitePan/data/`（368K）：`litepan.db`（+ `-wal`/`-shm`）、`secret.key`、`log/`、`backups/`、`upload_tasks/` 等 |
| **备份要点** | ⚠️ `litepan.db` 与 **`secret.key` 必须一起备份** —— 会话与加密数据依赖后者，只备 DB 会失效 |
| **回滚/停止** | `cd /root/LitePan && docker compose down`（删容器，**`data/` 保留**）；再次 `up -d` 即恢复 |
| **自愈** | `restart: unless-stopped`，宿主重启后自动拉起 |
| **镜像** | `ghcr.io/zhemed/litepan:v0.0.44`（本地已有，digest `sha256:a864057d…`） |

## 检查记录（trellis-check，2026-09-12）

**Step 1 变更识别**：仓库**零受跟踪文件改动**（仅本任务目录）；宿主侧新增容器 `litepan`、端口 `5211` 监听、`data/`+`mounts/` 目录（均 gitignored）。

**Step 3 项目质量门（本任务无代码改动，故质量门落在部署契约）**

| 检查 | 命令 | 结果 |
|---|---|---|
| 容器与端口 | `docker compose ps` / `ss -ltnp` | `Up`；`0.0.0.0:5211` + `[::]:5211` |
| 服务可用 | `/api/health`、`/`、`/api/auth/login`、`/api/public/system-config` | 200 / SPA / 登录成功 / `version=v0.0.44` |
| 启动日志 | `docker logs litepan` | `应用初始化完成` + `HTTP 服务已监听` |
| 持久化 | inode 比对 + restart + 只读读库 | 见下方第 1 条（四条独立证据） |
| 边界 | DSH 端口/PID、`git status`、`git check-ignore` | 全部未受影响 |

**Step 4 清单核对**

- 测试覆盖：**N/A** —— 本任务零代码改动，不新增函数或修复缺陷（部署验证即验收）。
- Spec 同步：**无新增经验需沉淀** —— 部署形态（compose + bind mount + `unless-stopped`）已在 README 与 compose 文件中记录；本任务未发现新约定。`docker-compose.fnos.yml` 的 tag 漂移属待决事项，未擅自改 spec。
- 范围纪律：未修改 `docker-compose.yml`、未加 healthcheck、未改端口、未改口令；改动逐条对应 D1–D5。
- 跨层一致性：**N/A**（无应用层改动）；但顺带在**部署形态**下复核了上一任务的成果 —— `/api/public/system-config` 在真实部署中返回 `v0.0.44` ✅。

**诚实记录：我自己的两个错误判据**

1. **持久化验证中我先后用了两个无效判据，均给出假阴性**：
   - ① 比对 `data/` **文件清单**前后一致 → 报 ❌。实际是应用重启后新建了 `cache/` 目录（运行期缓存），**新增文件不等于重置**。
   - ② 比对 **db+wal+shm 总字节数** → 报 ❌「疑似重置」（720816 → 262144）。**该指标对 SQLite WAL 模式根本无效**：WAL 的 683952B 含大量可复用帧，checkpoint 只把有效页写回主库再把 WAL 归零，总字节数下降是**正常且预期**的。
   - 最终改用四条与文件大小无关的判据（同一 inode、直接只读读库得 11 表 + 真实 pbkdf2 哈希、`secret.key` inode/mtime 未变、重启前的 session cookie 仍通过鉴权），结论才可靠。**教训：持久化验证不能用"文件字节数"这类会被正常重写行为扰动的指标。**
2. **应用未写入 `admin_username` 配置行**（`configs` 表只查到 `admin_password`）：属应用默认值行为，非部署问题，仅记录备查。
3. **UI 层未做浏览器级验证**（本机无无头浏览器）：以 API 实测 + SPA HTML 返回作为替代证据；界面上「当前版本」是否已显示 `v0.0.44`，需你在浏览器里最终确认一眼。
4. **待你决定的遗留项**：`docker-compose.fnos.yml:3` 的镜像 tag 仍为 `v0.0.31`（落后 13 个版本）。该文件为飞牛 NAS 专用、本机不加载；直接 bump 到 `v0.0.44` 会把 13 个版本的变更（含 STRM/crosstransfer 等功能的删除）一次性推给 NAS 用户，而本机无法验证 fnOS 环境 —— 故未擅自修改。
5. **镜像体积口径**（沿用上一任务的未决观察）：`v0.0.44` 未压缩 161MB / 压缩后 41.8MiB，与 README 首屏声称的 `118M` 不一致。
6. **本机默认口令 ≠ `AGENTS.md` 记录的 `123456`**：后者是**上一台机器**经授权重置后的值；本机为全新库，用的是首次启动默认值（`password_change_reason:default_credentials`）。你改密后，建议同步更新 `AGENTS.md` 的口令记录（那需要另建任务，因为它是规则文件）。
