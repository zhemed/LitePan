# fix-189-delete-accepted-semantics

## Goal

实施"受理即成功"（承接调查 09-10-investigate-delete-stale-folder-in-ui）：189 删除批任务受理后仅做**快速确认**（窗口 5s），确认超时不再报错——返回成功（189 异步完成）；显式失败（冲突/failedCount>0）与请求错误仍原样返回。`0.0.23` 发布。

## 方案

1. `waitBatchTask` 超时返回改用哨兵错误 `errBatchTaskTimeout`（消息文本不变）
2. 新增 `waitBatchTaskAccepted`：`errors.Is(err, errBatchTaskTimeout) → nil`（受理即成功）
3. `DeleteFiles` 的 DELETE 与 CLEAR_RECYCLE 两处等待改走 Accepted（窗口 5s）；MOVE/COPY（transferFiles 40s）**不动**——其超时更常代表真实失败，保持原语义
4. 测试：哨兵映射单测（超时→accepted / 其它错误→报错 / nil→nil）；全模块回归

## Acceptance Criteria

- [ ] vet 全绿 + go test ./... 零失败
- [ ] 0.0.23 三 tag + release + 部署三连
- [ ] 归档 + journal + push
