# design.md — 批次 2：熔断分组键

## 1. 改动边界

| 现状 | 目标 |
|---|---|
| 自动化路径不写 `BatchID` ⇒ 熔断计数直接跳过（`breaker.go:61-65`） | 自动化在一次运行内写入统一的 run 级 `BatchID/BatchName` |
| 熔断键 == `BatchID`（空则失效），计数与挑选各自比较 `BatchID` | 统一 `batchKeyOf(st)`：`batch:<id>` 或 `acct:<id>|target:<path>`，计数与挑选共用 |

**行为真正所在**：分组语义在 `internal/upload/breaker.go`；批次标识的**产生**在 `internal/automation/service_run.go`。两侧都改，缺一不可（只改自动化 → 其它空 id 路径仍无保护；只改熔断 → 自动化批次仍按「100 条分片」碎化）。

## 2. 契约

```go
// internal/upload/breaker.go
// batchKeyOf 返回熔断分组键：优先批次 id；无批次时退化为「账号 + 目标目录」。
func batchKeyOf(st *taskState) string
```

规则：
1. `strings.TrimSpace(st.BatchID) != ""` → `"batch:" + id`；
2. 否则 `TargetPath` 非空 → `fmt.Sprintf("acct:%d|target:%s", st.AccountID, st.TargetPath)`；
3. 否则 → `fmt.Sprintf("acct:%d", st.AccountID)`。

不变式：
- I1 计数（`observeBatchFailure`）与挑选（暂停剩余 pending）使用**同一个** `batchKeyOf`，不得再出现「一处用 id、一处用空判断」的分歧。
- I2 有 `batch_id` 的既有行为**完全不变**（键前缀不同但分组等价）。
- I3 熔断阈值仍为 5 次连续同因系统级失败；文件级错误不计入。

## 3. 数据流（自动化批次）

```
automation run 开始
  batchID = fmt.Sprintf("auto-%d-%d", ruleID, runStartUnix)
  batchName = fmt.Sprintf("定时备份 %s", ruleName)
  → 每个 CreateParams 带同一 batchID（跨 100 条分片、跨 mapping 均相同）
  → DB upload_tasks.batch_id/batch_name
  → 失败时 observeBatchFailure 用 batchKeyOf("batch:"+batchID) 计数
  → 第 5 次同因系统级失败 → 暂停同键的剩余 pending（提示语含「批次已自动暂停」）
```

## 4. 取舍

- **为何同时做两侧**：自动化侧解决「新建批次无标识」；熔断侧解决「历史/其它空标识任务无保护」。两者正交，缺一则仍有盲区。
- **为何退化键用「账号+目标目录」而非「账号」**：同一账号可能有多个目标目录的批次并存，按账号聚合会跨批次误暂停；目标目录是这批任务最自然的共同特征。目标目录为空时仍退化为账号级——此时任务无更细粒度特征，账号级保护优于无保护。
- **为何不改阈值/分类**：阈值与 systemic 判定是实现决定，与本次缺陷无关（范围纪律）。
- **回滚**：两个文件独立可回退；无数据迁移；历史任务 `batch_id` 仍为空但受退化键保护。

## 5. 风险

| 风险 | 处置 |
|---|---|
| 退化键过宽导致误暂停 | 只在「连续 5 次同因系统级失败」后触发；键含目标目录；测试覆盖「不同目录互不牵连」 |
| 自动化批次 id 每次运行不同，跨运行不累计 | 符合预期：熔断针对单次运行的连续故障；跨运行故障由账号级冷却/退避处理 |
| 批次名字符注入/敏感 | 仅规则名 + 时间，无路径敏感内容 |

## 6. 测试策略

- `breaker_test.go` 新增：空 id + 同账号同目标目录 → 5 次同因失败后剩余 pending 全部暂停（修复前失败）；空 id + 不同目标目录 → 不互相牵连；既有 `batch_id` 用例保持通过。
- `-race` 跑 `internal/upload/`。
- 自动化侧不做单测（依赖运行器），以代码审查 + 本地实测（触发一次 `local_upload` 运行，检查 DB `batch_id` 非空）作为验证。
