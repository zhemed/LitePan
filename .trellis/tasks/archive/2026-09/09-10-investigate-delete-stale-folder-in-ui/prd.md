# investigate-delete-stale-folder-in-ui

## Goal

调查 UI 删除 bulk-test-2000 后文件夹不消失、需强制刷新才消失的原因，定位根因并给出修复方案。只调查不改码。

## Acceptance Criteria

- [x] 根因闭环（research.md）：189 批删异步 + waitBatchTask 30s/70s 超时被当失败返回 502 → UI 报错保留条目 → 189 异步删完 → 强刷即消失。08:32:31 日志实证（502 + 超时错误，时间与操作吻合），云端实测删除实际成功
- [x] 次要因素排除：缓存失效链完好（parent_id 有传、InvalidateDirKeys 有执行）；前端成功路径本地移除正确
- [x] 修复建议："受理即成功"（快速确认窗口 3-5s，超时视为已受理；显式冲突/失败仍报错）
- [ ] 归档 + journal + push
