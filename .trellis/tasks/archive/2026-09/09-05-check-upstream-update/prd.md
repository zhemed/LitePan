# 检查上游作者Ponphil更新

## Goal

对比 origin(Ponphil/LitePan) 与本地 main，判断作者是否有新提交，评估是否需要同步

## Requirements

1. `git fetch origin` 获取作者最新分支/标签状态（只读，不合并、不改工作区）。
2. 对比维度：
   - `main/master` 分支 `HEAD` 是否领先本地（`git rev-list --left-right --count`）。
   - 作者近期提交时间线（近 10 条，含日期与 message）。
   - 作者是否有新 `tag`（与本地 `v0.0.12` 对比）。
   - 如有新增提交，分类评估：功能 / 修复 / 与本地精简方向是否冲突（STRM/离线/跨盘/WebDAV/通知等已删除模块相关的提交可直接忽略）。
3. 只读操作：不切换分支、不合并、不推送。

## Acceptance Criteria

- [ ] 已执行 `git fetch origin` 并记录输出
- [ ] 明确回答：作者有无更新（有→列出提交；无→给出双方 HEAD 与同步时间点）
- [ ] 给出同步建议：需要同步 / 不需要 / 需人工挑选 cherry-pick
- [ ] 全程只读，未改动工作区文件与分支状态

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
