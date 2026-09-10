# 设计：强制流程的规则化与工具化

## 1. 分层

| 层 | 载体 | 作用 | 生效范围 |
|---|---|---|---|
| 全局规则 | `~/.dsh/AGENTS.md` | 声明"Trellis 托管项目必须严格序列执行"的总原则 | 所有项目/会话 |
| 项目规则 | `/root/LitePan/AGENTS.md`（强制规则段） | 写明完整序列、每步完成判据、门禁命令 | 本项目所有会话 |
| 工具闸门 | `.trellis/scripts/flow_gate.py` | 在 start/archive 两个关键时刻做机器校验，拒绝不合规 | 本项目，可人工/钩子调用 |
| 技能 | `trellis-start` / `trellis-check` / `trellis-finish-work` | 提供步骤级操作规范（上下文载入、质量门清单、收尾） | 每次会话 |

## 2. flow_gate.py 设计

```
flow_gate.py pre-start <task>
  ├─ 定位 .trellis/tasks/<task>/（或 archive/<YYYY-MM>/<task>/）
  ├─ 校验 prd.md 存在
  ├─ 校验无未填充占位：正则 ^\s*-?\s*TBD\s*$ 与 "## Requirements" 段为空
  ├─ 读 task.json：dev_type/scope 非空且标记复杂 → 要求 design.md + implement.md
  ├─ 输出：下一步应执行 `task.py start <task>` 与技能提醒
  └─ 退出码 0/1

flow_gate.py mark-check <task>            # 通过质量门后打标
  └─ 写 .trellis/tasks/<task>/.check-passed（内容含时间戳与检查命令摘要）

flow_gate.py pre-archive <task>
  ├─ 校验 prd.md 的验收清单无未勾选（匹配 ^\s*- \[ \] ）
  ├─ 校验 .check-passed 存在且晚于最后一次代码提交/PRD 修改
  └─ 退出码 0/1，提示 archive 与 add_session 命令
```

判定"复杂任务"的规则：`task.json` 中 `scope` 非空且不等于 `lightweight`，或 `dev_type` 为 `feature`/`refactor`/`cross-layer`。未标注时给出警告但仍要求 design/implement（保守）。

## 3. 规则文本要点（写入两个 AGENTS.md）

- 强制序列（含 skill 载入与 check 门），并注明"任何写操作前必须已完成到 start 这一步"。
- 三条硬性禁止：先实施后补 PRD / 跳过 trellis-check / 用自建检查替代质量门。
- 门禁命令：start 前 `flow_gate.py pre-start`；check 后 `flow_gate.py mark-check`；archive 前 `flow_gate.py pre-archive`。
- 会话号口径：一律以 journal 记录为准。

## 4. 风险与回退

- 规则文件被 trellis update 覆盖？→ 项目规则只写托管块之外；全局规则在 `~/.dsh/` 不受影响。
- 门禁过严导致阻塞？→ 提供 `--force` 显式跳过参数（需在命令中写明原因字符串，写入 `.check-passed` 备注），保证可用性。
