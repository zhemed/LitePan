# Implementation Plan: 项目与工作区维护（D1–D6）

## Overview

严格按 design.md 的顺序执行：**D6（trellis update）最先**，其后 D1 → D2 → D4 → D5 → D3，最后统一过质量门并归档。

每个阶段都有**独立验证命令**与**回滚点**；任一 Gate 失败即停止推进（宿主清理 D3 放在最后，因为它不回滚且不影响其他项）。

前置：本机 `danger-full-access`，无需提权；审批已关闭，故**不请求任何 escalation**。

---

## Phase 0: 前置快照与备份

- [ ] 0.1 `git status --short` 记录基线（期望：仅本任务目录）
- [ ] 0.2 `cp .trellis/config.yaml /tmp/config.yaml.bak-$(date +%Y%m%d-%H%M%S)`；记录 `sha256sum` 与备份路径
- [ ] 0.3 记录升级前基线：`cat .trellis/.version`（期望 `0.6.16`）、`trellis --version`

**回滚点 R0**：备份路径与两处基线值写入本任务记录

---

## Phase 1: D6 — Trellis 升级 0.6.16 → 0.6.17（必须最先）

- [ ] 1.1 走保护路径：`trellis update --skip-all`（跳过"被用户改过"的文件，绝不用 `--force` 直接覆盖 `config.yaml`）
- [ ] 1.2 **验证门 G6a**：
      - `cat .trellis/.version` → `0.6.17`
      - 4 个模板文件确实变了：`git status --short .trellis/scripts .agents` 非空
      - `config.yaml` 与备份**逐字节相同**：`sha256sum -c` 或 `cmp`
- [ ] 1.3 **回退路径**（仅当 G6a 失败，即"跳过"把自动更新也一并跳过时）：
      `trellis update --force` → `cp /tmp/config.yaml.bak-<ts> .trellis/config.yaml` → 再验 `cmp` 一致
- [ ] 1.4 **验证门 G6b（升级后项目脚本自检）**：
      - `python3 ./.trellis/scripts/get_context.py` 正常输出
      - `python3 ./.trellis/scripts/task.py list` 能列出任务
      - `python3 ./.trellis/scripts/flow_gate.py pre-start 09-12-maintain-repo-and-workspace` 通过
      - `trellis --version` 不再提示 "update available"
- **回滚点 R1**：`trellis update --allow-downgrade`；升级会留 `.trellis/.backup-*` 目录

---

## Phase 2: D1 — 版本漂移三处同步

- [ ] 2.1 `.golangci.yml:4`：`go: "1.26.4"` → `go: "1.26.6"`
- [ ] 2.2 `Dockerfile:19`：注释 `go.mod 的 go 1.26.4` → `go.mod 的 go 1.26.6`
- [ ] 2.3 `.trellis/spec/backend/backend/quality-guidelines.md:17`：`go: "1.26.4"` → `go: "1.26.6"`
- [ ] 2.4 **验证门 G1**：全仓库 `grep -rn "1\.26\.4"`（排除 `node_modules`、`/tasks/archive/`、`/workspace/`）**零命中**
- **回滚点 R1'**：`git checkout -- .golangci.yml Dockerfile .trellis/spec/backend/backend/quality-guidelines.md`

---

## Phase 3: D2 — 消除 config.yaml 口令记录冲突

- [ ] 3.1 删除头部冲突行：`# 当前管理员：admin / admin（已落库 pbkdf2:sha256:600000$83f88b...，must_change:true）`
- [ ] 3.2 替换为指向单一权威，不含口令与 hash，例如：
      `# 管理员口令/凭据：以 AGENTS.md「项目强制规则」为唯一权威，此处不复记（避免双份记录漂移）`
- [ ] 3.3 **验证门 G2**：
      - `grep -n "83f88b\|admin / admin" .trellis/config.yaml` → 零命中
      - `diff <(grep -v "^#" /tmp/config.yaml.bak-<ts>) <(grep -v "^#" .trellis/config.yaml)` → **无非注释差异**（证明只动注释、未碰配置值）
- **回滚点 R2**：`cp /tmp/config.yaml.bak-<ts> .trellis/config.yaml`

---

## Phase 4: D4 — SIGPIPE 经验写入 spec/guides

- [ ] 4.1 新建 `.trellis/spec/guides/shell-script-guide.md`，覆盖三条实证坑：
      ① `pipefail` + `head`/`grep -q` 的 SIGPIPE(141) 假失败 → 改为"捕获变量 + `case` 子串匹配"
      ② 版本断言不要假设 `v` 前缀（`golangci-lint` 自报 `2.12.2` vs `Makefile` 钉 `v2.12.2`）→ 用去 `v` 数字串
      ③ 幂等与可复跑（`ln -sfn`、sha256 命中跳过下载、`[ -x ]` 已装跳过）
      每条附**真实复现命令**与出处（`09-12-setup-dev-environment`）
- [ ] 4.2 `.trellis/spec/guides/index.md` 的 "Available Guides" 表加一行指向新文件
- [ ] 4.3 **验证门 G4**：`grep -n "shell-script-guide" .trellis/spec/guides/index.md` 命中；新文件存在且非空
- **回滚点 R4**：`git rm`/删除新文件并还原 index.md

---

## Phase 5: D5 — 删除 6 张孤儿图片

- [ ] 5.1 `git rm docs/pictures/feature-strm.png docs/pictures/feature-strm-scrape.png docs/pictures/feature-crosstransfer.png docs/pictures/feature-organize.png docs/pictures/feature-automation.png docs/pictures/wechat-tip.png`
- [ ] 5.2 **验证门 G5**：`ls docs/pictures/` 仅剩 `banner.png`、`feature-browser.png`；`git status --short` 显示 6 个 `D`
- [ ] 5.3 确认无现役文档引用被删图：`grep -rn "feature-strm\|feature-crosstransfer\|feature-organize\|feature-automation\|wechat-tip" --include=*.md . | grep -v "/tasks/archive/"` → 零命中
- **回滚点 R5**：`git checkout HEAD -- docs/pictures/`（图片可从历史恢复）

---

## Phase 6: D3 — 宿主临时文件清理

- [ ] 6.1 `rm -f /tmp/go1.26.6.linux-amd64.tar.gz /tmp/go-dl.json /tmp/setup-env.log /tmp/setup-env-run2.log /tmp/setup-env-run3.log`
- [ ] 6.2 `rm -f .trellis/.runtime/update-check-*.marker`
- [ ] 6.3 **验证门 G3**：上列文件均不存在；且**未误删**他物 —— `ls /tmp | grep -c trellis` 仍 > 0（DSH/Trellis 运行时保留）
- **回滚点 R3**：无需回滚（仅缓存/日志；tarball 由 `setup-env.sh` 重下并校验 sha256）

---

## Phase 7: 质量门（维护后回归）

- [ ] 7.1 `cd /root/LitePan && make lint` → **必须仍 `0 issues.`**（关键：验证 `.golangci.yml` 的 `run.go=1.26.6` 可解析且不破坏规则）
- [ ] 7.2 `GOWORK=off go vet ./...` → exit=0
- [ ] 7.3 `GOWORK=off go test ./...` → 全包 `ok`
- [ ] 7.4 `cd web && npm run type-check` → exit=0
- [ ] 7.5 脚本自检复跑：`get_context.py` / `task.py list` / `flow_gate.py pre-start` 三连
- [ ] 7.6 `ss -ltnp` → `3080/3081` 仍由 DSH 监听
- [ ] 7.7 `git status --short` → 改动逐条对应 D1–D6，无越界文件

---

## Phase 8: 收尾归档

- [ ] 8.1 `skill trellis-check` 走查（覆盖核对 / spec 同步 / 范围纪律）
- [ ] 8.2 `flow_gate.py mark-check 09-12-maintain-repo-and-workspace --note "<质量门摘要>"`
- [ ] 8.3 勾选 `prd.md` 全部验收项（每条必须实际执行过）
- [ ] 8.4 `flow_gate.py pre-archive` 通过
- [ ] 8.5 `task.py archive 09-12-maintain-repo-and-workspace --skip-branch-validation`
- [ ] 8.6 `add_session.py`（会话号以 journal 为准）
- [ ] 8.7 手工 commit（archive 的 auto-commit 因路径搬迁已知会失败）→ `git push`

---

## Validation Commands（汇总，供 check 阶段复核）

```bash
# D1
grep -rn "1\.26\.4" --include=*.yml --include=Dockerfile --include=*.md . | grep -v node_modules | grep -v /tasks/archive/ | grep -v /workspace/
# D2
grep -n "83f88b\|admin / admin" .trellis/config.yaml
diff <(grep -v '^#' /tmp/config.yaml.bak-*) <(grep -v '^#' .trellis/config.yaml)
# D4
grep -n "shell-script-guide" .trellis/spec/guides/index.md
# D5
ls docs/pictures/
# D6
cat .trellis/.version && trellis --version
# D3
ls /tmp/go1.26.6.linux-amd64.tar.gz /tmp/go-dl.json /tmp/setup-env*.log /tmp/.trellis/.runtime/update-check-* 2>&1
# 回归
cd /root/LitePan && make lint
GOWORK=off go vet ./... && GOWORK=off go test ./...
cd web && npm run type-check
```

## Review Gates

| 门 | 位置 | 判据 |
|---|---|---|
| G6a | Phase 1.2 | `.version`=0.6.17 + 4 模板文件已变 + `config.yaml` 与备份逐字节相同 |
| G6b | Phase 1.4 | `get_context.py` / `task.py list` / `flow_gate.py pre-start` 三连可用 |
| G1 | Phase 2.4 | `1.26.4` 全仓库零命中 |
| G2 | Phase 3.3 | 冲突口令行零命中 + 非注释部分无差异 |
| G4 | Phase 4.3 | 新指南存在且索引已加行 |
| G5 | Phase 5.2/5.3 | 仅剩 2 张图 + 现役文档零引用 |
| G3 | Phase 6.3 | 目标残留已删且未误删 DSH/Trellis 运行时 |
| G7 | Phase 7 | `make lint` 0 issues + vet/test/type-check 全绿 + 改动可追溯 |

## Rollback

见 design.md → **Rollout / Rollback** 表。要点：每项独立可回滚；`config.yaml` 有 Phase 0 的逐字节备份；图片与 spec 均有 git 历史可恢复；Trellis 升级有 `.backup-*` 与 `--allow-downgrade`。

## Out of Scope

- 重建 `internal/api/web`（已核实无漂移）
- `/tmp` 中 DSH/Trellis 运行时占用（253M `trellis-research` 等）
- 改写归档任务文档与历史 journal
- 任何 Go/TS 源码、README、`go.mod`、`Makefile` 改动
- LitePan 部署（用户明确"等一下"）
