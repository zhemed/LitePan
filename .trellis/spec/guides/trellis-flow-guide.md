# Trellis 强制流程自检指南

> **用途**：每次会话开始、每次写操作前，用本清单确认自己在走"带门禁的流程"，而不是"边做边补手续"。
> 依据：`AGENTS.md`（项目强制规则）与 `~/.dsh/AGENTS.md`（全局硬约束 五）。

## 何时读

- 会话开始（尤其从上下文压缩恢复后）
- 准备执行任何写操作（代码/配置/DB/镜像/前端/规则文件）之前
- 归档任务之前

## 恢复会话时的第一件事（2026-09-10 违规根因）

**摘要里的流程描述 ≠ 规则原文**。压缩摘要曾把流程写成 `create → prd → start → implement → archive`，
漏掉了 `skill trellis-start` 与 `trellis-check` 两步，导致整段会话按"残缺规范"执行。

→ 恢复后必须重新读取 `AGENTS.md` 与 `.trellis/workflow.md` 再动手。

## 九步序列 + 每步判据

| 步 | 命令 | 判据 |
|---|---|---|
| 1 | `skill trellis-start` | 跑过 `get_context.py`（`--mode phase`/`--mode packages`）+ 读过相关 spec 索引 |
| 2 | `task.py create <name>` | 任务目录生成 |
| 3 | 写 `prd.md`（复杂任务加 `design.md`/`implement.md`） | **无 TBD**、验收标准可勾选；**早于 start** |
| 4 | `task.py set-scope <name> <scope>` | 标注轻/复杂（决定第 3 步产物） |
| 5 | `flow_gate.py pre-start <task>` → `task.py start` | 门禁通过 |
| 6 | 实施 + 质量门 | `go vet ./...`、`go test ./...`、`cd web && npm run type-check && npm run build` |
| 7 | `skill trellis-check` | 覆盖核对 + **spec 同步** + 范围纪律 + 跨层一致性 |
| 8 | `flow_gate.py mark-check` → `flow_gate.py pre-archive` → `task.py archive` | 验收全勾选 + check 标记存在 |
| 9 | `add_session.py` | 会话号**以 journal 为准** |

## 四条硬性禁止

- ✗ 先实施后补 PRD（PRD 若与归档同秒写入，就是事后记录，规划门已失效）
- ✗ 用自建 vet/test/build 三连顶替 `trellis-check`（会漏 spec 同步/覆盖核对/范围纪律）
- ✗ 跳过 `trellis-start` 的上下文与 spec 索引载入
- ✗ 未标注 scope、复杂任务缺 `design.md`+`implement.md` 就 start

## 机器门禁

```bash
python3 ./.trellis/scripts/flow_gate.py pre-start   <task>   # 拒 TBD / 复杂任务缺产物
python3 ./.trellis/scripts/flow_gate.py mark-check  <task>   # 质量门通过后打标
python3 ./.trellis/scripts/flow_gate.py pre-archive <task>   # 拒未勾选验收 / 缺 check 标记
# 显式跳过（写明原因，记录进 .check-passed）：
... --force "原因字符串"
```

## 自检问句

1. 我的 PRD 是**动手之前**写的吗？（看 `prd.md` mtime 是否早于首次代码改动）
2. 这个任务算复杂吗？复杂的话 `design.md`/`implement.md` 在吗？
3. `trellis-check` 的六步我过了几步？spec 有没有需要同步的经验？
4. 我改了验收标准之外的文件吗？（范围纪律）
5. 我汇报的会话号和 journal 一致吗？
