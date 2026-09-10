# implement.md — 批次 1 执行计划

## 顺序

1. `internal/upload/state.go`
   - 新增 `beginCooldownWait(taskID string, seconds int) bool`（锁内判定 + 迁移 + 值快照 + 持久化 + broadcast）
   - `patch()` 内 `snap := st` → `snap := *st`（值拷贝）并传 `&snap`
   - 验证：`GOWORK=off go build ./...`
2. `internal/upload/worker.go`
   - 冷却分支：删除 `if !m.canCooldownWait(taskID) { return false }` + 无条件 `m.patch` 组合，改调 `beginCooldownWait`；守卫失败 `return false`
   - 等待结束仍 `return m.canCooldownWait(taskID)`
   - 验证：`GOWORK=off go vet ./internal/upload/`
3. `internal/upload/cooldown_pause_test.go`（新增）
   - 守卫拒绝矩阵 5 例 + 放行 1 例 + 竞态一致性 2 例（pause↔beginCooldownWait 两个顺序）
   - 先写测试、确认「守卫拒绝」用例在旧实现下会失败（记录为回归证据）
   - 验证：`GOWORK=off go test ./internal/upload/ -run 'Cooldown' -v`、`go test -race ./internal/upload/`
4. spec 同步 `.trellis/spec/backend/backend/upload-task-api.md` §8 第 3 条（暂停优先 + 原子守卫 + 值快照）
5. 质量门：`go vet ./...`、`go test ./...`、`cd web && npm run type-check && npm run build`
6. 版本 `0.0.34`（README/docker-compose）→ 构建推送镜像 ×3 → `git tag v0.0.34` + release → 本地容器重建 + 健康/登录/任务列表三连
7. 收尾：`skill trellis-check` → `mark-check` → `pre-archive` → `archive --skip-branch-validation` → `add_session.py` → `git push github main`

## 验证命令

```bash
GOWORK=off GOTOOLCHAIN=local go vet ./... && GOWORK=off GOTOOLCHAIN=local go test ./...
GOWORK=off GOTOOLCHAIN=local go test -race ./internal/upload/
git diff --name-only   # 期望：internal/upload/**、.trellis/spec/**、README.md、docker-compose.yml
```

## 回滚点

- 步骤 1/2 各自可独立回退；整体 `git revert`（无数据迁移、无配置变更）。
- 若本地部署后出现「冷却后任务不再重试」：立即回退步骤 2 的调用点（守卫放行条件疑似过严）。

## 范围纪律（不做）

- 不改 `persist.go` 恢复逻辑、不改并发/节流参数、不改前端、不改驱动、不动 0.0.32 冷却日志抑制。
