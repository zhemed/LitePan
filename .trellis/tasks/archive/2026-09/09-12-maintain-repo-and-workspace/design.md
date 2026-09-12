# Design: 项目与工作区维护（D1–D6）

## Overview

六项维护彼此独立，但存在**一条硬依赖**：`trellis update`（D6）会改写 `.trellis/scripts/` 并可能询问 `config.yaml`，因此必须最先执行，之后的 D2（编辑 `config.yaml`）才能成为最终态。其余五项互不干扰，按"配置类 → 文档类 → 文件类 → 宿主类"分组执行，便于中途失败时定位。

改动面刻意收窄到 **3 个配置/文档文件 + 1 个新增 spec 文件 + 1 个索引行 + 6 张图片删除 + 1 段 config.yaml 注释 + 宿主 `/tmp` 清理**，不触碰任何 Go/TS 源码。

## Boundaries

| 层 | 改 | 不改 |
|---|---|---|
| **Lint 配置** | `.golangci.yml:4` `run.go` `1.26.4`→`1.26.6` | 其余 lint 规则、`depguard` 配置、`allow-parallel-runners` |
| **构建文档** | `Dockerfile:19` 注释 `1.26.4`→`1.26.6` | `Dockerfile` 的 `FROM`/`ENV`/`RUN` 任何指令（`FROM` 早已是 `golang:1.26.6-bookworm`） |
| **Trellis spec** | 新增 `spec/guides/shell-script-guide.md`；`spec/guides/index.md` 加一行 | `spec/backend/**`、`spec/web/**` 的既有规范内容（仅 `quality-guidelines.md:17` 的版本号同步） |
| **Trellis 元数据** | `.trellis/config.yaml` 头部注释一行；`.trellis/.version`（由 update 写入） | `config.yaml` 其余全部内容；`.trellis/tasks/**`、`.trellis/workspace/**`（历史不改写） |
| **Trellis 框架文件** | 由 `trellis update` 自动更新的 4 个模板文件 | 项目自研脚本 `flow_gate.py`、`get_context.py`、`task.py`（非模板托管，升级不动它们） |
| **文档资源** | 删除 6 张孤儿 PNG | `banner.png`、`feature-browser.png`；`README.md` 正文（不含图片引用的重写） |
| **宿主 /tmp** | 我方安装残留 3 类文件 + `.runtime` 陈旧 marker | `/tmp/trellis-*`、`/tmp/node-compile-cache`、`/tmp/plain-proj`、`/tmp/dump-*.yml`、`/tmp/gh-u.json` |
| **工程源码** | 无 | `internal/`、`drivers/`、`pkg/`、`cmd/`、`web/`、`go.mod`、`go.sum`、`Makefile`、`README.md` |

## Data Flow（执行顺序与依赖）

```
[0] 前置快照
    git status 基线 / 备份 .trellis/config.yaml → /tmp/config.yaml.bak-<ts>
         │
[D6] trellis update                                   ← 必须最先（会改写 .trellis/scripts/）
    ├─ 期望：自动更新 4 个模板文件 + .trellis/.version → 0.6.17
    ├─ config.yaml 属"modified by you" → 用跳过方式处理，绝不用 --force 直接盖
    ├─ 若"跳过"把自动更新也一并跳过 → 回退路径：--force 全量升级，再从备份恢复 config.yaml
    └─ 升级后自检：get_context.py / flow_gate.py pre-start / task.py list 三连可用
         │
[D1] 版本漂移三处同步改写（.golangci.yml + Dockerfile + spec/quality-guidelines.md）
         │
[D2] .trellis/config.yaml 头部过期管理员注释 → 指向 AGENTS.md 单一权威
         │
[D4] 新增 spec/guides/shell-script-guide.md + index.md 加行
         │
[D5] git rm 6 张孤儿图片
         │
[D3] 宿主清理：rm /tmp 三类残留 + rm .trellis/.runtime/update-check-*.marker
         │
[质量门] make lint（关键：验证 .golangci.yml 新 go 设定）
         + go vet + go test + web type-check
         + 脚本自检（get_context / flow_gate / task.py list）
         │
[收尾] trellis-check → mark-check → 勾选验收 → pre-archive → archive → add_session
```

## Key Decisions

### KD1 为什么 `trellis update` 必须最先

`trellis update` 会改写 `.trellis/scripts/common/active_task.py` 与 `task_store.py`（任务状态读写的核心），若先做 D2 再升级，`config.yaml` 会处于"刚被改过"的状态，升级的 3-way 判定更难解释。先升级、后编辑，可让 D2 成为明确的最终态，也让"升级后脚本仍可用"的验证覆盖到全部后续写操作。

### KD2 `config.yaml` 的保护策略：跳过优先，force 兜底

`--dry-run` 已明确 `config.yaml` 是"Modified by you (need your decision)"。选择顺序：

1. `trellis update --skip-all`（跳过用户改动文件，期望自动更新仍生效）
2. 校验：`.trellis/.version == 0.6.17` 且 4 个模板文件已变 且 `config.yaml` 与备份逐字节相同
3. 若步骤 2 失败（自动更新被一并跳过）→ 回退 `trellis update --force`，然后 `cp` 备份回 `config.yaml`，并再次逐字节校验

不选"直接 `--force`"作为首选，是因为那会在无提示的情况下把项目自定义文件重置为模板态，属于不可逆的信息丢失。

### KD3 `run.go` 改 `1.26.6` 是否安全

golangci-lint 要求 `run.go` 不得高于构建该二进制的 Go 版本；本机 `golangci-lint` 正是用 `go1.26.6` 构建（`--version` 已证），故设 `1.26.6` 安全。且 `go.mod` 亦为 `go 1.26.6`。**验证方式不靠推理而靠实测**：改完立刻 `make lint`，必须仍 `0 issues.`

### KD4 spec 指南的落点与语言

放 `.trellis/spec/guides/`（与 `code-reuse-thinking-guide.md`、`cross-layer-thinking-guide.md`、`trellis-flow-guide.md` 同级），并在 `index.md` 的 "Available Guides" 表加行 —— 该表是 spec 索引入口，不加行则新指南不会被后续会话发现。

语言按 `spec/backend/backend/index.md` 末行约定 "spec is in **English**" 用英文，但保留必要中文术语与真实命令，便于复现。

### KD5 图片删除的边界

判据（两条同时满足）：① 当前仓库无任何引用（README/其他现役文档）或对应功能已从代码删除；② 不属于"功能仍在"的截图。据此：

- 删：`feature-strm.png`、`feature-strm-scrape.png`（STRM 已整体删除）、`feature-crosstransfer.png`（跨盘秒传已删）、`feature-organize.png`（目录整理已删）、`feature-automation.png`（自动化仅剩 `local_upload`）、`wechat-tip.png`（零引用）
- 留：`banner.png`（README 首图）、`feature-browser.png`（文件浏览功能仍在）

接受代价：归档任务文档的引用变悬空。**不改写归档文档** —— 归档是当时状态的历史记录，为清理而回改历史会破坏其证据价值。

## Compatibility

- **`.golangci.yml` `run.go`**：`1.26.6` 与 `golangci-lint` 构建用 Go、`go.mod`、`Dockerfile` 的 `FROM`、`GOTOOLCHAIN=local` 四处一致，消除"CI 用 1.26.4 语义分析、实际用 1.26.6 编译"的隐性不一致。
- **`trellis update`**：项目自研的 `flow_gate.py` 不是模板托管文件（不在 `--dry-run` 的自动更新列表），故门口径不变；`.trellis/spec/` 属"用户数据，保留"，D4 新增的指南不会被升级冲掉。
- **`/tmp` 清理**：`setup-env.sh` 已是幂等脚本（tarball sha256 命中即跳过、`[ -x ]` 已装即跳过），删除缓存后重跑该脚本会重新下载并校验，**不破坏可复现性**；已安装的 `/usr/local/go` 不受影响。
- **图片删除**：`docs/pictures/` 不被代码或构建引用（无 `go:embed`、无 `vite` 引用），删除不影响构建产物。
- **DSH 共存**：全程不触碰 `3080/3081` 与其服务。

## Tradeoffs

- **一任务六项 vs 拆三个任务**：六项都很小且共享同一批质量门，拆开会把 `make lint`+`go test` 三连重复三遍、并产生三套归档与 journal 记录；合并为一个 `multi-deliverable` 任务更省，验收项仍**逐条独立可验**。代价是单任务 diff 面稍宽，靠 `git status` 逐条回溯 PRD 来约束。
- **跳过 vs 覆盖 `config.yaml`**：跳过保住项目自定义（含中文强制流程说明），代价是若模板有实质变化则 `config.yaml` 稍旧；但该文件的价值在项目的自定义头部，模板默认值差异影响极小。
- **删图 vs 留图给上游对比**：删了少 1.04M 并从"已删功能"的视觉残留中清理干净；代价是失去与上游 `Ponphil/LitePan` 对比时的截图素材 —— 但 git 历史完全可恢复，可逆。

## Rollout / Rollback

| 项 | 回滚方式 |
|---|---|
| D1 三处版本号 | `git checkout -- .golangci.yml Dockerfile .trellis/spec/backend/backend/quality-guidelines.md` |
| D2 config.yaml | `cp /tmp/config.yaml.bak-<ts> .trellis/config.yaml` |
| D4 新增 spec | `git rm .trellis/spec/guides/shell-script-guide.md` 并还原 index.md |
| D5 图片 | `git checkout <本任务改动前的 HEAD> -- docs/pictures/` |
| D6 Trellis 升级 | `trellis update --allow-downgrade` 或 `.trellis/.backup-*`（升级会留备份目录） |
| D3 宿主清理 | 无需回滚（仅删缓存与日志；tarball 可由脚本重下） |

## File Map

| 路径 | 动作 | 归属 |
|---|---|---|
| `.golangci.yml` | 改 1 行（`run.go`） | D1 |
| `Dockerfile` | 改 1 行（注释） | D1 |
| `.trellis/spec/backend/backend/quality-guidelines.md` | 改 1 行（示例版本号） | D1 |
| `.trellis/config.yaml` | 改头部注释块（删除冲突口令行，改指向 AGENTS.md） | D2 |
| `.trellis/spec/guides/shell-script-guide.md` | 新增 | D4 |
| `.trellis/spec/guides/index.md` | 加 1 行指南表条目 | D4 |
| `docs/pictures/{feature-strm,feature-strm-scrape,feature-crosstransfer,feature-organize,feature-automation,wechat-tip}.png` | 删除 6 个 | D5 |
| `.trellis/.version`、4 个模板文件 | 由 `trellis update` 写入 | D6 |
| `/tmp/go1.26.6.linux-amd64.tar.gz`、`/tmp/go-dl.json`、`/tmp/setup-env*.log` | 删除 | D3 |
| `.trellis/.runtime/update-check-*.marker` | 删除 | D3 |
| `internal/**`、`web/**`、`go.mod`、`Makefile`、`README.md` | **零改动** | — |
