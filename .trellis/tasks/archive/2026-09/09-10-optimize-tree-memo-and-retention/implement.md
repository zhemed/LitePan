# 执行步骤

1. [ ] 前端：TaskPanel 行记忆化 + 批次节点签名记忆化 + 上限保护 + 统计导出
2. [ ] 前端验证：esbuild 编译纯逻辑 → Node 断言（未变化复用/变化重建/上限）
3. [ ] 后端：registry 新增两键（intSpec，组 system）
4. [ ] 后端：Manager RetentionConfig + selectRetentionVictims（纯函数）+ pruneRetainedTasks + retentionLoop
5. [ ] 后端：app 装配注入配置闭包（settings-backed）
6. [ ] 后端：单测覆盖选择逻辑（超期/超量/非终态/禁用）
7. [ ] 质量门：go vet/test + web type-check/build
8. [ ] 0.0.28 三 tag + release + 部署三连 + 实测（设置 API 新键、默认不误删）
9. [ ] 勾选验收 → mark-check → pre-archive → archive → add_session → push
10. [ ] spec 同步：upload-task-api.md 增补保留策略段（跨层契约变更）
