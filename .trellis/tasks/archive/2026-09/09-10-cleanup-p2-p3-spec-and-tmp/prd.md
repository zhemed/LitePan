# cleanup-p2-p3-spec-and-tmp

## Goal

完成残留盘点报告的 **P2 + P3**：旧 DB 备份归档、补 0.0.27 窗口化任务 API 的跨层契约 spec、flow_gate 短名支持、/tmp 杂物清理。来源：`09-10-investigate-residual-maintenance` research.md §P2/§P3。

## Requirements

1. **P2-DB**：`data/litepan.db.bak.1788077861`（229K，8-30 旧备份）归档进 `data/backups/`（保留而非删除，命名带出处与日期）
2. **P2-spec**：新增 `.trellis/spec/backend/backend/upload-task-api.md`，按 update-spec 的 **7 段式**记录 0.0.24~0.0.27 的跨层契约（窗口化列表/汇总/SSE counts、批量 pause/resume、冷却等待语义），并在 backend `index.md` 登记
3. **P3-gate**：`flow_gate.py` 支持短任务名（唯一后缀匹配），扩展自测覆盖该路径
4. **P3-tmp**：清理本会话在 `/tmp` 产生的杂物（**仅限本会话创建的文件**，非本会话文件保留并说明）

## Constraints

- 不改动应用代码（无版本发布）；工具与文档变更除外
- 只删自己创建的 /tmp 文件；不确定归属的一律保留
- 不删任何 DB 备份（只做归档重命名）

## Acceptance Criteria

- [x] 旧备份归档为 data/backups/legacy-20260830-before-0022.db（229K 保留），backups 目录 3 份清单已列
- [x] upload-task-api.md 落盘（7 段齐全，覆盖窗口化/汇总/SSE/批量控制/冷却/受理即成功）+ index.md 登记
- [x] flow_gate 短名支持 + 自测 6/6（唯一命中/歧义拒绝/全名回归/archive 短名）+ 实测短名调用通过
- [x] 清理 11 个探针文件 + 2 个 fixture 目录（/tmp 65M→38M）；13:00-14:21 的 19 个非本会话文件保留并说明
- [x] 质量门 + 门禁 + 归档 + journal + push
