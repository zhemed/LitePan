# 修正 NAS compose 过期 tag 与目录分歧、清理三处无法核实的体积数字

## Goal

收尾上一轮部署暴露的两项遗留：

1. **`docker-compose.fnos.yml` 双重过期**：镜像 tag 停在 `v0.0.31`（落后 13 个版本），且它的数据目录是 `/vol1/1000/docker/**litepango**/` 而 README 快速开始写的是 `/vol1/1000/docker/**litepan**/` —— 照 README 抄的人会落到完全不同的目录。
2. **`118M` 这个数字从来就没对上过**：按用户决策**直接去掉**，不填新值。

## Background（实测证据，2026-09-12）

### 体积数字的取证

| 版本 | `docker images`（未压缩） | 压缩层合计（= 用户实际下载量） |
|---|---|---|
| `v0.0.1`（当年写的"稳定基线"） | **178MB** | **46.0 MiB** |
| `v0.0.44`（当前） | **161MB** | **41.8 MiB** |

- `118M` **与两者都不符** —— 既不是未压缩也不是下载量，说明它从一开始就不是实测值。
- 取证方法：`gh api` 分页查到 `v0.0.1` 仍在 GHCR（tag 组 `v0.0.1`/`0.0.1`/`v0.5.2-Beta`），`docker pull` 后分别用 `docker images` 与解析 amd64 子 manifest 的层 size 求和测得。测完已 `docker rmi` 清理。
- 附带事实：镜像**随功能精简变小了**（v0.0.1 的 178MB → v0.0.44 的 161MB），所以"越新越大"的直觉在这里不成立。

### 118M 的全部出现处（3 处，排除 `.git`/`node_modules`/归档/workspace/backup）

| 位置 | 原文 |
|---|---|
| `README.md:9` | `**115 · 天翼 · 本机 · 一个界面 · 118M**` |
| `AGENTS.md:62` | `**版本基线**：\`0.0.1\` 即稳定基线（\`118M 3驱动\`，…）` |
| `.trellis/config.yaml:9` | `# 版本基线：0.0.1 即稳定基线（118M 3驱动，…）` |

### `litepango` 的全部出现处（2 处，均在 fnos compose 的 volumes 段）

`docker-compose.fnos.yml:8` 与 `:9`。

### 一并核验为**准确**、故保持不变的声明

- `仅 3 驱动` ✅（`drivers/{115_Open,189Cloud,LocalFs}` + `template`）
- 驱动 ID `115_open` / `189_cloud` / `localfs` ✅ 与代码一致
- `Vue ^3.5.41`、`Go 1.26.6`、镜像 tag `v0.0.44` ✅

## Requirements

### D1 `docker-compose.fnos.yml` 对齐（tag + 目录）

- `image`: `ghcr.io/zhemed/litepan:v0.0.31` → `ghcr.io/zhemed/litepan:v0.0.44`
- 两处数据目录：`/vol1/1000/docker/litepango/` → `/vol1/1000/docker/litepan/`（与 README 快速开始统一，按用户决策以 `litepan` 为准）
- **其余字段一律不动**：`network_mode: "host"`、`restart: always`、`privileged: true`、`pid: "host"`、`devices: /dev/fuse`、`TZ=Asia/Shanghai`、`container_name` —— 这些是 fnOS 场景的既定选择，本任务不重新设计
- **已知影响（如实记录，不在本任务处理）**：NAS 用户从 `v0.0.31` 直接跳到 `v0.0.44`，会一次性跨过 13 个版本，其中包含 **STRM、跨盘秒传、多驱动等功能的删除**。本机无法验证 fnOS 环境，故只改文件、不做 NAS 侧验证；用户如需分步升级，应改回中间版本自行验证。

### D2 去掉无法核实的体积数字（3 处同批）

- `README.md:9`：`**115 · 天翼 · 本机 · 一个界面 · 118M**` → `**115 · 天翼 · 本机 · 一个界面**`
- `AGENTS.md:62`：去掉 `118M ` 前缀（`\`118M 3驱动\`` → `\`3驱动\``），该句其余部分（0.0.1 基线、`ghcr.io/zhemed/litepan:0.0.1` 已推、`git tag v0.0.1`、递增规则、不跳 `1.0.0`）**完整保留**
- `.trellis/config.yaml:9`：同上处理
- **按用户决策，不填入任何新数字**（不写 42M / 161MB，也不写"约 xx"）

## Constraints

- **不改代码**：不动 `internal/`、`drivers/`、`web/` 任何源码；不动运行中的 `litepan` 容器
- **不改 README 其它内容**：`README.md` 的 diff 必须只有第 9 行一处
- **不改 fnos compose 的其它字段**（见 D1）
- **三处 `118M` 必须同批清理**：只改 README 会在 `AGENTS.md`/`config.yaml` 留下同源错误数字，将来又会被抄回 README —— 这正是本仓库反复出现的"多处副本必然漂移"问题
- **保留仍准确的声明**：`3驱动`、驱动 ID、Vue/Go 版本、镜像 tag 不得误删或改动
- 改 `AGENTS.md`（规则文件）与 `.trellis/config.yaml` 属规则文件写操作，故本任务先建任务并 `start` 后再生效（符合项目强制规则）
- 不修改归档任务与历史 journal

## Acceptance Criteria

- [x] `grep -rn "118M"`（排除 `.git`/`node_modules`/`/tasks/`/`/workspace/`/`.backup-`）零命中 —— 实测**零命中**
- [x] `grep -rn "litepango"` 全仓库（同上排除）零命中 —— 实测**零命中**
- [x] `docker-compose.fnos.yml` 的 image 为 `ghcr.io/zhemed/litepan:v0.0.44`，两处卷路径均为 `/vol1/1000/docker/litepan/...` —— 实测 image 已 bump、两处路径已对齐 README
- [x] `docker compose -f docker-compose.fnos.yml config` 解析通过 —— 实测 `--quiet` 通过，文件仍是合法 compose
- [x] `README.md` 首屏为 `**115 · 天翼 · 本机 · 一个界面**`；`git diff --stat README.md` 仅 1 行变动 —— 实测 `1 file changed, 1 insertion(+), 1 deletion(-)`，diff 只有第 9 行
- [x] `AGENTS.md` 该句仍完整包含 `0.0.1` 基线、`ghcr.io/zhemed/litepan:0.0.1`、`git tag v0.0.1`、递增规则、`不跳 1.0.0`，且已无 `118M` —— 五项要素逐一 grep 确认 ✅；系统重载 AGENTS.md 亦确认新内容生效
- [x] `.trellis/config.yaml` 该行仍含 `版本基线` 与 `3驱动`，且已无 `118M`；非注释部分零差异 —— 实测与上轮备份 `diff`（去注释行）**零差异**，仅注释变化
- [x] 仍准确的声明未被误改 —— 实测 `仅 3 驱动`、`115_open`、`189_cloud`、`localfs`、`Vue 3.5.41`、`v0.0.44` 均在位；`go.mod` 仍为 `go 1.26.6`
- [x] 运行中的容器未受影响 —— 实测 `litepan | Up 8 minutes`，`/api/health` HTTP 200
- [x] `git status --short` 改动逐条对应 D1–D2，无越界文件 —— 实测 4 个受跟踪文件（`docker-compose.fnos.yml`、`README.md`、`AGENTS.md`、`.trellis/config.yaml`），合计 **6 insertions / 6 deletions**

## Notes

- Scope 标注 `lightweight`：纯文档/配置修正，共 4 个文件、改动行数极少，且无技术方案分歧（用户已就两处决策点给出选择），故 PRD-only，不另写 `design.md` / `implement.md`。
- 本任务**不**做：fnOS 环境实测、NAS 分步升级指引、README 其它内容重写、补充新的体积指标。
- 与上一任务的衔接：`09-12-deploy-litepan-local-5211` 的检查记录里把 `118M` 与 fnos tag 列为"待决遗留"，本任务即其处置闭环。

## 检查记录（trellis-check，2026-09-12）

**Step 1 变更识别**：4 个受跟踪文件，合计 **6 insertions / 6 deletions**（`docker-compose.fnos.yml` 3 处、其余各 1 处）。零代码改动。

**Step 2 规范对齐**：本任务属文档/配置修正，未触及任何分层契约；改动前后均保持了 compose 文件的合法性与 README 的结构。

**Step 3 项目质量门**

| 检查 | 结果 |
|---|---|
| `docker compose -f docker-compose.fnos.yml config --quiet` | 通过（文件仍合法） |
| `docker compose config --quiet`（本机在用的那份） | 通过（未被误动） |
| `curl /api/health` + `docker compose ps` | HTTP 200 / `litepan Up 8 minutes`（运行中服务未受本次改动影响） |
| Go / 前端质量门 | **N/A** —— 零代码改动，未新增函数或修缺陷 |

**Step 4 清单核对**

- 测试覆盖：**N/A** —— 纯文档与 compose 配置，无可测行为变更；`config --quiet` 作为 compose 的语法回归。
- Spec 同步：**无需更新** —— 本任务是把两处**声明**对齐事实，未产生新约定。上一任务已在 `api-layering.md` 记下"发版需同步 README/compose 镜像 tag"的约定；本次进一步暴露"compose 文件有多个、容易漏改最后一个"这一操作性教训。
- 范围纪律：未动 README 其它内容、未动 fnos compose 其它字段、未动代码与容器。
- 跨层一致性：N/A。

**诚实记录：发现与判断**

1. **`118M` 的取证方式**：为判断该填什么数字，我从 GHCR 分页查出 `v0.0.1` 仍在（tag 组 `v0.0.1`/`0.0.1`/`v0.5.2-Beta`），`docker pull` 后分别实测未压缩 **178MB** 与压缩层合计 **46.0 MiB** —— `118M` 与两者都不符，且 v0.0.44 比 v0.0.1 更小（161MB / 41.8 MiB，因功能精简）。取证用的镜像测完已 `docker rmi` 清理。
2. **因此该数字不是"过时"而是"从未准确"**：这是我把判断交给用户而非自行替换的原因。用户选择**去掉**，本次即不填任何新值。
3. **三处必须同批清理**：只改 README 会在 `AGENTS.md` 与 `.trellis/config.yaml` 留同源错误 —— 而"多处副本必然漂移"正是本仓库这一整轮连续修的主题（版本号三副本、口令两副本、镜像 tag 三文件）。本次一并清零，并保留仍准确的 `3驱动`。
4. **fnos tag 的已知影响已记录未处理**：NAS 用户从 `v0.0.31` 跳到 `v0.0.44` 会跨过 13 个版本（含 STRM/跨盘秒传/多驱动删除）。本机无法验证 fnOS 环境，故只改文件；如需分步升级，用户应自行改为中间版本验证。
5. **未见其它同类漂移**：`docker-compose.yml`、`README.md` 的镜像 tag 均已是 `v0.0.44`；驱动 ID、Vue/Go 版本等声明经逐一核验仍准确。
