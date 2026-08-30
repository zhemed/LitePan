# Record versioning 0.0.1 stable

## Goal

将用户于 `2026-08-30` 明确的 **“`0.0.1` 即稳定基线，后续 `0.0.2` 递增，不跳 `1.0.0`”** 持久化到 `AGENTS.md` 与 `.trellis/config.yaml` 注释，使后续所有 `Trellis` 会话与 `ghcr` 标签均按 `0.0.x` 递增。

## Requirements

- **版本基线**：`0.0.1`（`118M 3驱动`，`ghcr.io/zhemed/litepan:0.0.1` 已推，`git tag v0.0.1`）即稳定基线，**非** `1.0.0`
- **递增**：`fix` → `0.0.2`，`feat` → `0.0.3`，`breaking` 亦 `0.0.4`，**不跳 `1.0.0`**，仅用户显式说“发 `1.0`”时再 `1.0.0`
- **落盘**：`AGENTS.md` 的 `项目强制规则` 段追加 `版本基线：0.0.1 稳定，后续 0.0.2 递增`，`.trellis/config.yaml` 顶部注释同步

## Constraints

- 仅改 `AGENTS.md` 与 `.trellis/config.yaml` 的注释，不改 `drivers` 代码
- `grep -c "0.0.1.*稳定" AGENTS.md` >=1 且 `grep -c "0.0.2.*递增" AGENTS.md` >=1
- `grep -c "1.0.0" AGENTS.md` == 0（除历史 `spec` 外）

## Acceptance Criteria

- [ ] `AGENTS.md` 含 `0.0.1.*稳定` 且 `0.0.2.*递增`
- [ ] `AGENTS.md` 无 `1.0.0` 的跳变描述
- [ ] `.trellis/config.yaml` 顶部含 `0.0.1 稳定` 注释

## Notes

- 用户明确“我们现在0.0.1就是稳定基线呀，谁说得1.0.0才是呀？”，本次纠正 `SemVer` 的 `1.0.0` 误导
