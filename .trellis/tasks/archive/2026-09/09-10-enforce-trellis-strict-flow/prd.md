# enforce-trellis-strict-flow

## Goal

把 trellis 强制流程从"靠自觉"变为"规则约束 + 工具拦截"：写入全局规则与项目规则，并新增阶段门脚本；通过加载技能使本会话即生效。

## Requirements

1. **全局规则**（`~/.dsh/AGENTS.md`）：新增"Trellis 托管项目严格执行"条目——凡 Trellis 托管项目（`.trellis/` 存在）的写操作，必须按强制顺序执行；禁止"先实施后补 PRD"、禁止跳过质量门。
2. **项目规则**（`/root/LitePan/AGENTS.md` 强制规则段，位于 Trellis 托管块之外）：写清完整强制序列与每步的完成判据：
   `skill trellis-start（载入上下文/spec 索引）→ task.py create → prd(+复杂任务 design/implement) → task.py set-scope → task.py start → 实施 → skill trellis-check → task.py archive → add_session.py`
3. **阶段门脚本** `.trellis/scripts/flow_gate.py`（新增）：
   - `pre-start <task>`：拒绝 `prd.md` 缺失或仍含 TBD 占位；复杂任务（scope 标注）拒绝缺 `design.md`/`implement.md`；输出后续步骤提醒
   - `pre-archive <task>`：拒绝验收标准存在未勾选项；拒绝缺少 check 记录标记
   - `mark-check <task>`：在通过 `trellis-check` 的项目检查后写入标记（供 pre-archive 校验）
   - 退出码：0 通过 / 1 拒绝（信息明确）
4. **加载**：本会话载入 `trellis-start`、`trellis-check`、`trellis-finish-work` 技能与强化后的规则文件，确保立即生效。

## Constraints

- 不修改 `.trellis/` 托管块内部（会被 trellis update 覆盖）；项目规则写在托管块之外。
- 不改变已有任务档案内容（除本任务）。
- 脚本仅依赖 Python 标准库，零第三方依赖。

## Acceptance Criteria

- [x] 全局规则含 Trellis 严格执行条目（~/.dsh/AGENTS.md 第五节，含压缩恢复特别要求）
- [x] 项目规则含九步序列表 + 每步判据 + 四条硬性禁止 + 门禁命令（AGENTS.md 强制规则段）
- [x] `flow_gate.py` 三子命令实现，自测 8/8（应通过/应拒绝/标记后放行/--force 四类）
- [x] 本会话已加载 trellis-start/trellis-check/trellis-finish-work/trellis-update-spec 技能；两个规则文件均已被系统重新载入生效
- [x] 质量门通过（vet/测试/类型检查/构建）+ spec 同步（新增 flow 指南并入索引）+ 门禁标记 + 归档 + journal + push
