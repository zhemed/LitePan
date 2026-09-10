# 执行步骤

1. [ ] `executeUpload` 返回值改为 `requeue bool`（所有 return 点补 false，冷却分支按语义返回）
2. [ ] `runTask` 增重入循环（releaseSlot → requeue → acquireRunSlot）
3. [ ] 新增 `canCooldownWait` 判定并接入冷却分支（含等待期间取消的处理）
4. [ ] 单测：冷却 → requeue=true 且 pending；暂停态 → false 且保持 paused；等待期间取消 → false 不覆盖
5. [ ] 全量门禁（vet/test/check:memo/type-check/build）
6. [ ] 0.0.30 三 tag + release + 部署三连
7. [ ] 部署后实测：既有孤儿任务恢复；人为构造冷却验证 30 秒后自动重试；暂停不被覆盖
8. [ ] 勾选验收 → mark-check → pre-archive → archive → add_session → push
