# 修正 drivers/template 包注释指向不存在的 README

## Goal

`drivers/template/driver.go:1` 的包注释写「按 **README** 实现」，但该目录下**只有 6 个 `.go` 文件、没有 README** —— 按这段注释去新建驱动的人会先去翻一个不存在的文件。本任务把该指引改为指向**真实存在且持续维护**的文档，并保留其余准确部分。

## Background（2026-09-12 实测）

注释原文：

```go
// Package template 是新驱动脚手架：复制本目录为 drivers/<名>/，改包名与 Config.Name，按 README 实现后再于 all.go 空导入（勿注册本包）。
package template
```

**逐处核实**（不整段推翻，只修确实错的那处）：

| 注释中的说法 | 实测 | 判定 |
|---|---|---|
| 「复制本目录为 `drivers/<名>/`」 | 与 spec 的 `cp -r drivers/template drivers/FooCloud` 一致 | ✅ **准确，保留** |
| 「改包名与 `Config.Name`」 | 与 spec 第 1、2 步一致 | ✅ **准确，保留** |
| **「按 README 实现」** | `ls drivers/template/` = `auth.go` `config.go` `driver.go` `models.go` `ops.go` `transport.go`，**无 README**；全仓库 grep `template/README` **仅此一处** | ❌ **错误，本任务修的就是它** |
| 「于 `all.go` 空导入」 | `drivers/all.go` **存在**；spec 第 4 步亦要求「add import in `drivers/all.go`」 | ✅ **准确，保留** |
| 「勿注册本包」 | `internal/driver/registry.go` 的注册表**无 template 条目**；template 仅被 `internal/auth/oauth_integration_test.go` 空导入 | ✅ **准确，保留** |

**真实且维护中的文档在哪**：`.trellis/spec/backend/backend/driver-development.md` 的
「Adding a New Driver `FooCloud`」一节 —— 含完整四步（copy template → 实现 `driver.go` 的
`Config()`/`GetAddition()` → 填 `auth.go`/`transport.go`/`ops.go`/`upload.go`（含 `DelayController.Gate` 用法）
→ 注册进 `drivers/all.go` + `registry.go`）。

## Requirements

### D1 只改这一行注释，指向真实文档

- 把「按 README 实现」替换为指向 `driver-development.md`「Adding a New Driver」的指引
- **保留**其余三处已核实为真的指引（复制目录、改包名与 `Config.Name`、`all.go` 空导入、勿注册本包）
- 新注释须**自足**：即使不打开 spec 也能看懂"复制 → 改标识 → 实现接口 → 空导入"的顺序；spec 路径作为**深入指引**附在后面

### D2 不得引入新的悬空引用

- 新注释中出现的每一个文件路径都必须**实际存在**（在改动后立即逐一验证）
- 不得指向 `.trellis/` 之外的、也不得指向未提交的产物

## Constraints

- **只改 `drivers/template/driver.go` 的包注释**：不改该文件的任何代码、不改 template 目录其它文件
- **不改 `drivers/all.go`**（它是对的）
- **不改 spec**（`driver-development.md` 本身准确，是被引用方而非错误方）
- **不新增 README 文件**：用户已明确选择"改注释"而非"补文档"
- **不顺手做其它清理**：本任务与 P4 保留决策一致，template 整体继续保留
- 质量门：`make lint` / `go vet` / `go test` 须全绿（注释改动不应影响任何行为，但需验证没手滑改坏代码）

## Acceptance Criteria

- [x] 包注释**不再出现**「README」字样 —— 实测 `grep -n "README" drivers/template/driver.go` **零命中**
- [x] 新注释指向 `driver-development.md` 的「Adding a New Driver」，且该文件与章节**确实存在** —— 实测该文件存在，且 `grep -n "Adding a New Driver"` 命中 **`driver-development.md:69`**
- [x] 新注释中提到的文件路径（`drivers/all.go`）**实际存在** —— 实测 `test -f drivers/all.go` 通过
- [x] 其余三处准确指引仍在 —— 实测四项逐一命中：`复制本目录为 drivers/<名>/`、`改包名与 Config.Name`、`all.go 空导入`、`勿注册 template 自身`
- [x] **只改注释** —— 实测 `git diff` 中非注释行改动数 = **0**（1 行删除 + 3 行新增，全部以 `//` 开头）
- [x] `gofmt -l drivers/template/driver.go` 无输出 —— 实测**符合 gofmt**
- [x] 质量门全绿 —— 实测 `make lint` **0 issues**、`go vet` exit=0、`go test` **27 包 ok**
- [x] **`deadcode` 仍为 7、`unused` 仍为 0** —— 实测两项均**未变**（本任务不应改变死代码基线）
- [x] `git status --short` 仅该文件 + 本任务目录 —— 实测无越界改动

## Notes

- Scope 标注 `lightweight`：单文件单行注释修正，无技术方案分歧，故 PRD-only。
- 来源：P3/P4 详情说明时顺带发现的缺陷（`drivers/template` 包注释指向不存在的 README）；用户选择"改注释"。
- 本任务**不**做：新增 README、删除 template（P4 已判定保留）、修正 `offline_handoff` 的命名漂移、处理 `refresh-auth` 端点。

## 检查记录（trellis-check，2026-09-12）

**Step 1 变更识别**：单文件 `drivers/template/driver.go`，**1 行删除 + 3 行新增**，**非注释行改动 = 0**（`git diff` 逐行核验）。零代码改动。

**Step 3 项目质量门（全部实测）**

| 检查 | 结果 |
|---|---|
| `make lint` | `0 issues.` |
| `GOWORK=off go vet ./...` | exit=0 |
| `GOWORK=off go test ./...` | **27 包 ok** |
| `gofmt -l drivers/template/driver.go` | 无输出（符合 gofmt） |
| 死代码基线复核 | `deadcode` **仍 7**、`unused` **仍 0**（本任务不应改变基线，实测未变） |

**Step 4 清单核对**

- 测试覆盖：**N/A** —— 纯注释修正，无可测行为变更；`go test` 全绿作为"没手滑改坏代码"的回归证据。
- Spec 同步：**无需更新** —— 本次是让代码注释**指向**既有 spec，被引用方（`driver-development.md`）本身准确，未产生新约定。
- 范围纪律：只改 1 个文件的注释；未动 `drivers/all.go`、未动 spec、未新增 README。
- 跨层一致性：N/A。

**诚实记录**

1. **逐处核实后才改，未整段推翻**：注释里共四处指引，实测**三处为真**（`复制目录`、`改包名与 Config.Name`、`all.go 空导入`、`勿注册本包`），**只有「按 README 实现」是假的**（该目录只有 6 个 `.go` 文件）。故只替换那一处，其余原样保留 —— 避免把对的也改错。
2. **修的是"指向不存在的文件"，而非"文档缺失"**：用户明确选择改注释而非补 README；且真实文档早已存在（`driver-development.md:69` 的「Adding a New Driver」含完整四步，包括 `DelayController.Gate` 用法与注册位置），实际只是注释指向错了地方。
3. **全仓库只有这一处**指向该不存在的 README（`grep -rn "template/README"` 仅命中此文件），无同源副本需要一并修。
4. **P4 决策未受影响**：本次修正的是注释，`drivers/template` 整体按 P4 判定继续保留（它仍是 spec 记载的驱动创建流程载体与 `TestOAuthDriversUseUnifiedGuard` 的测试替身）。
