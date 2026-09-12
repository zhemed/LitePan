# 维护：修版本漂移、清理陈旧配置与临时文件、Trellis 升级 0.6.17

## Goal

对拉取到本机的 `LitePan` 仓库与其 Trellis 元数据做一次**有据可查的维护**（用户全选授权六项）：消除配置/文档与 `go.mod` 的版本漂移、消除自相矛盾的口令记录、清理本次安装留下的临时文件、把上一轮踩到的脚本陷阱沉淀进 spec、清理已删功能的孤儿文档图片，并把 Trellis 框架从 `0.6.16` 升级到 `0.6.17`。

全部改动均来自 2026-09-12 的只读巡查结论，不做无依据的"顺手整理"。

## Background

只读巡查（2026-09-12）确认的事实：

- **仓库状态健康**：工作区干净、与 `origin/main` 同步；`v0.0.1` → `v0.0.43` tag 齐全（当前版本有 tag）；`.trellis/tasks/` 无残留、无缺 `task.json` 的任务。
- **`go:embed` 前端产物无漂移**（排查后排除）：唯一"未配对"的前端提交 `75be52b` 删除 23 个死文件，逐个反查当前 `web/src` 引用数为 **0** → 死代码删除不改变构建产物，故已提交的 `internal/api/web` 仍准确。**本任务不重建 embed**。
- **`1.26.4` 全仓库仅 3 处**（`grep -rn` 排除 `node_modules`/归档/workspace 后）：`.golangci.yml:4`、`.trellis/spec/backend/backend/quality-guidelines.md:17`、`Dockerfile:19`。而 `go.mod` 是 `go 1.26.6`、`Dockerfile:15` 的 `FROM` 已是 `golang:1.26.6-bookworm`。
- **口令记录冲突**：`.trellis/config.yaml` 头部注释写 `admin / admin`（`pbkdf2:sha256:600000$83f88b...`，`must_change:true`），而 `AGENTS.md:60` 的权威记录是 `admin / 123456`（`pbkdf2:sha256:600000$aa9764...`，2026-09-09 经授权重置，且明确"此前 08-30 记录的 admin/admin 已作废"）。
- **`trellis update --dry-run` 的实际影响面**：只自动更新 4 个模板文件（`.trellis/scripts/common/active_task.py`、`.trellis/scripts/common/task_store.py`、`.agents/skills/trellis-session-insight/SKILL.md`、`.agents/skills/trellis-session-insight/references/cli-quick-reference.md`）；`.trellis/config.yaml` 因被项目改过而需人工决定；`.trellis/spec/`、`.trellis/tasks/`、`.trellis/workspace/`、`.trellis/.developer` 属"用户数据，保留"。
- **临时文件归属已分清**：`/tmp` 380M 中我方安装残留约 66M（`go1.26.6.linux-amd64.tar.gz` 64M、`go-dl.json` 2.2M、3 个 `setup-env*.log`）；其余（`trellis-research` 253M、`trellis-prefix` 53M、`trellis-node-compile-cache` 等）属 DSH/Trellis 运行时，**不在本任务范围**。另 `/tmp/gh-u.json` 经检查只是 GitHub 公开用户资料 JSON（`login/id/avatar_url`），**不含任何 token 或密钥**，无安全处置需求。
- **图片引用现状**（共 8 张，1.7M）：README 只引用 `banner.png`；`feature-browser.png` 对应功能**仍存在**；其余 6 张或指向**已删除功能**（STRM、crosstransfer、目录整理、automation 大部分已删），或**零引用**（`wechat-tip.png`），且全部只被 `.trellis/tasks/archive/` 的历史文档提及。

## Requirements

### D1 版本漂移修复（3 处，必须同步改齐）

- `.golangci.yml:4`：`go: "1.26.4"` → `go: "1.26.6"`（功能性：golangci-lint 据此判定分析用的语言版本）
- `Dockerfile:19`：注释中的 `go 1.26.4` → `go 1.26.6`（`FROM` 已是 1.26.6，注释过期）
- `.trellis/spec/backend/backend/quality-guidelines.md:17`：内嵌示例中的 `go: "1.26.4"` → `go: "1.26.6"`（**spec 同步** —— 只改配置不改文档会立刻制造新的漂移）

### D2 消除 `.trellis/config.yaml` 的口令记录冲突

- 删除头部注释里重复且已过期的「当前管理员：`admin/admin` + 旧 hash」整行，替换为**指向单一权威**的表述（管理员口令以 `AGENTS.md` 为准，不在第二处复记口令与 hash）
- 理由：两处独立记录必然再次漂移；口令/凭据类信息应只存一处，且该处已有明确的变更流程（改密须先建 Trellis 任务 + `ask_user_question`）
- **保留** `config.yaml` 其余项目自定义内容，不覆盖、不重置

### D3 清理临时文件与陈旧运行时标记

- 删除 `/tmp/go1.26.6.linux-amd64.tar.gz`（64M）—— `setup-env.sh` 会按 `go.dev/dl/?mode=json` 重新下载并校验 sha256，删除不影响可复现性
- 删除 `/tmp/go-dl.json`（2.2M）—— 同上，脚本会重新拉取
- 删除 `/tmp/setup-env.log`、`/tmp/setup-env-run2.log`、`/tmp/setup-env-run3.log` —— 关键结论已固化在归档任务 `09-12-setup-dev-environment/prd.md` 的检查记录中
- 删除 `.trellis/.runtime/update-check-dsh_session-<旧 hash>.marker` —— 属旧会话遗留的运行时标记
- **不动**：`/tmp/trellis-*`、`/tmp/node-compile-cache`、`/tmp/plain-proj`、`/tmp/dump-*.yml`、`/tmp/gh-u.json`（非本任务产物）
- **不动** `/tmp` 之外的任何宿主状态（Docker/Go/golangci-lint 已装好的工具链）

### D4 把 SIGPIPE 陷阱沉淀进 spec

- 新增 `.trellis/spec/guides/shell-script-guide.md`，并在 `.trellis/spec/guides/index.md` 的指南表加一行
- 内容必须包含三条**本项目真实踩过的**坑（证据来自 `09-12-setup-dev-environment`）：
  1. `set -o pipefail` 下 `cmd --version | head -1` 与 `cmd | grep -q`：下游提前退出使上游收到 SIGPIPE(141)，管道整体判为失败 → **假失败**；正确做法是"先捕获变量，再用 `case` 做子串匹配"
  2. 版本断言不要带 `v` 前缀假设：`golangci-lint` 自报 `2.12.2` 而 `Makefile` 钉 `v2.12.2`，带 `v` 的断言会误判 → 用去 `v` 的数字串比较
  3. 幂等与可复跑：`ln -sfn`、sha256 命中即跳过下载、`[ -x ]` 已装即跳过 —— 使脚本可安全重复执行
- spec 为英文惯例（见 `spec/backend/backend/index.md` 末行 "spec is in **English**"），正文用英文、可保留必要中文术语

### D5 清理孤儿文档图片

- 删除 6 张（共约 1.04M）：`feature-strm.png`(37K)、`feature-strm-scrape.png`(457K)、`feature-crosstransfer.png`(95K)、`feature-organize.png`(62K)、`feature-automation.png`(70K)、`wechat-tip.png`(344K)
- **保留**：`banner.png`（README 引用）、`feature-browser.png`（对应功能仍存在）
- 已知代价（明确接受）：`.trellis/tasks/archive/` 中的历史文档对这 6 张图的引用将成为**悬空引用** —— 归档记录按当时状态保留，**不改写历史文档**；图片可由 git 历史恢复

### D6 Trellis 框架升级 0.6.16 → 0.6.17

- 运行 `trellis update`，**保留项目自定义的 `.trellis/config.yaml`**（不覆盖）
- 升级后必须验证项目自研脚本仍可用：`get_context.py`、`flow_gate.py`、`task.py current/list`
- 升级会更新 `.trellis/.version` 到 `0.6.17` 与 4 个模板文件

## Constraints

- **不做范围外改动**：不重建 `internal/api/web`（已核实无漂移）；不动 `web/`、`internal/`、`drivers/`、`pkg/`、`cmd/` 任何 Go/TS 源码；不改 `go.mod`/`go.sum`/`Makefile`/`README.md`；不动 DSH（`3080/3081`）
- **`.golangci.yml` 改动必须复跑 `make lint` 验证**，确认新语言版本设定下仍 `0 issues.`
- **`trellis update` 顺序与安全**：先备份 `config.yaml` → 用"跳过用户改动文件"的方式执行 → 校验 4 个模板文件与 `.version` 已更新且 `config.yaml` 内容未变；若自动更新被一并跳过，则回退为 `--force` + 从备份恢复 `config.yaml`
- **图片删除只删上述 6 张**，不做"顺手"重命名或目录重组
- **`/tmp` 清理只删 D3 列明的条目**，不 `rm -rf /tmp/*`
- 归档任务文档与历史 journal **一律不改写**
- 全部改动需通过质量门后才归档；宿主改动与仓库改动都要在验收中留痕

## Acceptance Criteria

- [x] 配置/源码/现役文档中 `1.26.4` **零命中**（精确口径：`grep -rn "1\.26\.4"` 再排除 `node_modules`、`/tasks/archive/`、`/workspace/`、本任务规划文档）—— 实测零命中；剩余命中仅在归档任务 `design.md` 与本任务规划文档（描述该变更本身，属历史记录，按约束不改写）；三处目标已确认为 `1.26.6`
- [x] `cd /root/LitePan && make lint` 仍输出 `0 issues.` —— 实测 `0 issues.` exit=0（证明 `run.go=1.26.6` 可被 golangci-lint 2.12.2 正常解析）
- [x] `.trellis/config.yaml` 中不再出现 `admin/admin` 与旧 hash `83f88b`，改为指向 `AGENTS.md` 作为口令单一权威；其余项目自定义内容保持不变 —— 实测冲突行零命中，且 `diff` 非注释部分**零差异**（只动注释，配置值逐字节未变）
- [x] `.trellis/spec/guides/shell-script-guide.md` 存在，且 `spec/guides/index.md` 的指南表已含其条目 —— 实测 130 行新指南 + index.md:27 已加行；含 SIGPIPE、`v` 前缀、版本可取性、幂等四条
- [x] `docs/pictures/` 仅剩 `banner.png` 与 `feature-browser.png` 两张；`git status` 显示 6 个删除 —— 实测剩余 2 张、6 个 `D` 已暂存；现役文档零引用（仅归档历史文档悬空，按约束不改写）
- [x] `.trellis/.version` 内容为 `0.6.17`；`trellis --version` 无 "update available" 提示 —— 实测 `.version=0.6.17`，`trellis --version` 直出 `0.6.17` 无提示
- [x] 升级后项目脚本自检通过：`get_context.py`、`flow_gate.py pre-start`、`task.py list` —— 实测三连 ✅，且 `task.py list` 正确标注 `09-12-maintain-repo-and-workspace … <- current`（无跨会话错绑）
- [x] `/tmp` 中 `go1.26.6.linux-amd64.tar.gz`、`go-dl.json`、`setup-env*.log` 均已不存在；`.trellis/.runtime/update-check-*` 陈旧 marker 已不存在 —— 实测 5 个文件全删、marker 0 个；且未误删 DSH/Trellis 运行时（`/tmp/trellis-*` 仍 4 个目录，`/tmp` 380M→314M，释放 66M）
- [x] `cd /root/LitePan && GOWORK=off go vet ./...` 与 `GOWORK=off go test ./...` 仍全绿 —— 实测 vet exit=0、go test exit=0 全包 `ok`
- [x] `cd /root/LitePan/web && npm run type-check` 仍通过 —— 实测 `vue-tsc -b` exit=0
- [x] `ss -ltnp` 确认 `3080/3081` 仍由 DSH 监听 —— 实测 2 个监听在位
- [x] `git status --short` 的改动**逐条可追溯到本 PRD 的 D1–D6**，无越界文件 —— 实测 17 文件变更（+75/−19）+ 2 项未跟踪（新指南、本任务目录）；`internal`/`drivers`/`pkg`/`cmd`/`web`/`go.mod`/`go.sum`/`Makefile`/`README.md` 改动数均为 **0**

## Notes

- Scope 标注 `multi-deliverable`（D1–D6 六个可独立验证的交付物），故按复杂任务补齐 `design.md` + `implement.md` 后再 `start`。
- **执行顺序有依赖**：D6（trellis update）必须**最先**做 —— 它会改写 `.trellis/scripts/`，若先改 `config.yaml` 再升级，可能被升级流程当作"用户改动"再次询问；先升级再编辑可让我的 D2 改动成为最终态。
- **本任务不解决**：`internal/api/web` 与 `web/src` 的（已排除的）漂移、`/tmp` 的 DSH 运行时占用、`.trellis/workflow.md` 的模板版本细化差异（升级会自动处理）。
- 图片删除是**可逆**操作（`git checkout <删除前的 commit> -- docs/pictures/<file>` 或从历史恢复），故风险可控。

## 检查记录（trellis-check，2026-09-12）

**Step 1 变更识别**：17 个受跟踪文件（+75/−19）+ 2 项未跟踪（`spec/guides/shell-script-guide.md`、本任务目录）。源码零改动：`internal`/`drivers`/`pkg`/`cmd`/`web`/`go.mod`/`go.sum`/`Makefile`/`README.md` 改动数均为 0。

**Step 2 规范对齐**：任务产物 `prd.md` + `design.md` + `implement.md` 均已读回核对；`get_context.py --mode packages` 的 `backend` / `web` 索引本会话已载入；本次改动未触及任何包的分层契约。

**Step 3 项目质量门（全部实测）**

| 检查 | 命令 | 结果 |
|---|---|---|
| Lint | `make lint` | `0 issues.` exit=0（**关键**：`run.go` 由 1.26.4 改 1.26.6 后仍解析正常） |
| Vet | `GOWORK=off go vet ./...` | exit=0 |
| Test | `GOWORK=off go test ./...` | exit=0，全包 `ok` |
| Type-check | `cd web && npm run type-check` | exit=0 |
| 脚本自检 | `get_context.py` / `task.py list` / `flow_gate.py pre-start` | 三连 ✅ |

**Step 4 清单核对**

- 代码质量：lint / type-check / test 全绿；本任务零源码改动，故无 debug 日志、无抑制告警。
- 测试覆盖：**N/A** —— 未新增/修改任何 Go/TS 函数、未修复任何产品缺陷。新增的 `shell-script-guide.md` 是规范文档；新增的删除项是二进制资源。
- **Spec 同步：本次已执行（与上一任务不同）** —— ① `1.26.4→1.26.6` 同步进 `spec/backend/backend/quality-guidelines.md`（否则配置与文档立刻二次漂移）；② 新增 `spec/guides/shell-script-guide.md` 并登记进索引，把 `09-12-setup-dev-environment` 的两个真实假失败沉淀为可复用规范。
- 跨层一致性（Step 5）：**N/A** —— 未触及 API/Service/Store/Driver/UI 任一层数据流。
- 范围纪律：`git status` 逐条可追溯 D1–D6，无越界文件；未顺手重命名/重组目录；未重建 embed（已先证无漂移）。

**诚实记录：偏离与代价**

1. **`config.yaml` 的处置按设计走了"跳过"路径**：`trellis update --skip-all` 报告 `Auto-updated: 4 file(s)` + `Skipped: 1 file(s)`（即 `config.yaml`），`cmp` 证明该方法下 `config.yaml` 与备份**逐字节相同**，故 `design.md` 中的 `--force` 回退路径**未被触发**。
2. **验收项 1 的口径在执行中收紧**：初始 grep 会命中**本任务自己的规划文档与归档任务 `design.md`**（它们描述该变更本身）。已把口径明确为"排除 `node_modules`/`/tasks/archive/`/`/workspace/`/本任务规划文档"，并按新口径复测为零命中。这是澄清口径，不是放宽标准。
3. **归档文档的悬空图片引用是明知的代价**：11 个归档文件仍引用被删的 6 张图。按 Constraints「归档记录不改写」保留，图片可由 git 历史恢复。
4. **`.trellis/.version` 与 `.trellis/.template-hashes.json` 的改动不是手写**，由 `trellis update` 自动写入，归 D6。
5. **未触碰** `/tmp` 的 DSH/Trellis 运行时（`trellis-research` 253M、`trellis-prefix` 53M 等 4 个目录），故 `/tmp` 仅从 380M 降至 314M —— 释放的 66M 全部来自本任务列明的 5 个文件。
