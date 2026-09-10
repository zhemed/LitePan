# implement.md — 批次 2 执行计划

## 顺序

1. `internal/upload/breaker.go`
   - 新增 `batchKeyOf(st *taskState) string`
   - `observeBatchFailure`：`batchID := ...` 改为 `key := batchKeyOf(st)`；计数 map 与「挑选剩余 pending」均改用 `key`
   - 验证：`GOWORK=off go build ./...`
2. `internal/automation/service_run.go`
   - 在 `local_upload` 动作进入 mapping 循环前生成 run 级 `batchID` / `batchName`（`fmt.Sprintf("auto-%d-%d", ruleID, time.Now().Unix())`）
   - 把两者写入 `upload.CreateParams{...}`（第 481-497 行的字面量）
   - 验证：`GOWORK=off go vet ./internal/automation/`
3. `internal/upload/breaker_key_test.go`（新增）
   - 空 id + 同账号同目标目录 → 熔断暂停剩余 pending（**先确认修复前失败**）
   - 空 id + 不同目标目录 → 不牵连
   - `batchKeyOf` 三态单测（有 id / 有目标 / 都无）
   - 验证：`GOWORK=off go test ./internal/upload/ -run 'Breaker|BatchKey' -v`、`-race`
4. spec 同步 `.trellis/spec/backend/backend/upload-task-api.md` §8 第 4 条
5. 质量门：`go vet ./...`、`go test ./...`、`cd web && npm run type-check && npm run build`
6. 版本 `0.0.35` → 镜像构建推送 ×3 → `git tag v0.0.35` + release → 本地容器重建 + 三连验证
7. 可选实测：本地触发一次自动化 `local_upload`（用小目录），确认新任务 `batch_id` 非空（只读 DB 查询）
8. 收尾：`skill trellis-check` → 门禁 → archive → add_session → push

## 验证命令

```bash
GOWORK=off GOTOOLCHAIN=local go vet ./... && GOWORK=off GOTOOLCHAIN=local go test ./...
GOWORK=off GOTOOLCHAIN=local go test -race ./internal/upload/
git diff --name-only   # 期望：internal/upload/breaker.go、internal/automation/service_run.go、测试、spec、版本文件
```

## 回滚点

- 步骤 1/2 独立回退；整体 `git revert`。
- 若本地实测发现自动化批次 id 生成异常（如每次分片不同），回退步骤 2 并复核 `flush()` 边界。

## 范围纪律（不做）

- 不改熔断阈值与 `systemicFailure` 分类、不改前端、不改 `CreateBatch` 契约、不动冷却日志抑制（0.0.32）。
