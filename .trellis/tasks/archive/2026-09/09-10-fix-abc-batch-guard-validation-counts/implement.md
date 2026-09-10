# 执行步骤（分批）

## 批 A（高危，先修）
1. [ ] `CreateParams.BatchTaskTotal` + `manager.createTask` 写入 `result.batch_task_total` + `retainBatchRootMetadata` 保留该键
2. [ ] `api/local_upload.go` 传入本批文件总数
3. [ ] `retention.go`：`selectRetentionVictims` 跳过 owned 批次根任务
4. [ ] `delete.go`：根删除前增加 `batch_task_total` 判据（缺失/不等 → 跳过根删除）
5. [ ] 测试：复现测试改为断言"不删根"；新增正向用例（完整选择 + 有总量 → 仍可删根）；保留策略跳过用例
6. [ ] 提交（批 A）

## 批 B（校验）
7. [ ] `deleteFiles` 过滤空白 ID + 全空 400；测试覆盖
8. [ ] 提交（批 B）

## 批 C（计数）
9. [ ] `uploadTaskTotals.ts` 纯函数 + store/徽标/导航接入
10. [ ] 前端校验脚本断言（paused 场景）；type-check/build
11. [ ] 提交（批 C）

## 收尾
12. [ ] 全量门禁 → 0.0.29 三 tag + release + 部署三连 → 实测（删除不再误删目录、徽标正确）
13. [ ] spec 同步（保留策略段补充"跳过 owned 批次根"与 batch_task_total 契约）
14. [ ] mark-check → 勾选验收 → pre-archive → archive → add_session → push
