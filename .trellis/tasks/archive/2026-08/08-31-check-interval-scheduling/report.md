# 报告：间隔执行“下次执行”为何落到次日

**任务**：`08-31-check-interval-scheduling`  
**截图**：`36a465 704x149` `00:05 起 每 1 小时 从指定时间开始按间隔轮询` + `7aab37 274x154` `下次执行 2026-09-01 00:05`  
**基线**：`service_schedule.go 154 computeIntervalStartRunAt` + `service.go 187 computeNextRun(now)`  
**方法**：只读 `grep/cat` + 复现推演

---

## 一、复现

**用户配置**：`start_time="00:05" interval_hours=1`，当前 `now≈2026-08-31 14:00`（截图次日为 `2026-09-01`，说明当天 `08-31` 已过 `00:05`），预期 `当天下一档 15:05 或 14:05`，实际 `次日 00:05`。

**当前 `computeIntervalStartRunAt` 逻辑（154-161）**：

```go
h,m := parseClock(start_time) // 0,5
next := Date(base.Year, base.Month, base.Day, h,m,0,0, base.Location) // 当天 00:05
if next.After(base) { return next } // 仅当 00:05 仍在未来才返回当天
nextDay := next.AddDate(0,0,1)
return Date(nextDay.Year, nextDay.Month, nextDay.Day, h,m,0,0, ...) // 否则次日 00:05
```

- `base=2026-08-31 14:00` → `next=00:05` 不 `After` → 返回 `2026-09-01 00:05`（截图一致）
- `base=2026-08-31 00:00` → `00:05 After` → 返回 `当天 00:05`（正确）
- `base=2026-08-31 01:00` → 同样次日 `00:05`（应为 `01:05`）

**对照用例**（`service_test.go 121` 现有测试仅覆盖锚点）：
- `base 12:00 start 13:00 interval1 → 13:00` ✅（锚点未来）
- `base 12:00 start 01:00 interval1 → 次日01:00` ✅（锚点已过，按现逻辑次日）

**缺口**：*未考虑同日内 `start + n*interval` 的中间档*。用户 `每1小时` 预期是 `00:05,01:05,...,23:05` 当天 24 档，但代码只认 `首档 00:05`。

---

## 二、根因

**初始 `NextRunAt` 计算错误**，而 **递进 `advanceIntervalRunAt` 正确**：

- `advanceIntervalRunAt 169-177`：
  ```go
  candidate := current.Add(interval) // 如 current=00:05 → 01:05
  if sameLocalDay(candidate, current) { return candidate } // 同日递进正确
  // 跨天才回到次日 00:05
  ```
  此段对 *已运行后* 的下一档是正确的。

- `computeIntervalStartRunAt` 却 **未递进**，直接跳次日锚点，导致 *首次* 调度即等到次日。

**触发路径**：
1. `service.go:187 NextRunAt = computeNextRun(..., time.Now())` 创建时即 `次日`
2. `service_schedule.go:68 computed := computeNextRun(..., now)` 次日
3. `scheduleOnce` 中 `if !rule.NextRunAt.IsZero() { nextRun = rule.NextRunAt }` 取到次日值
4. `if nextRun.After(now) { continue }` 次日 > now，跳过执行，直接显示次日

**时区**：`wallClockTime` 取 `time.Local`，若容器 `TZ=Asia/Shanghai` 与浏览器一致则无偏差；若不一致会额外偏移，但非此截图主因（截图已 `次日` 而非 `时差`）。

---

## 三、应有行为（预期）

**语义**：“`00:05 起每1小时`” = 锚点 `00:05`，间隔 `1h`，当天 `00:05 … 23:05`，次日再 `00:05`。

- `now 14:00` → `next = 14:05`（若 `14:05` 已过则 `15:05`）——**当天**
- `now 00:00` → `00:05` 当天
- `now 23:10` → `次日 00:05`（当天已无档）

**修复思路**（`computeIntervalStartRunAt`）：

```go
anchor := Date(base.Year, base.Month, base.Day, h,m,0,0, loc)
if anchor.After(base) { return anchor }
 // 从 anchor 起，每 interval 递进直到 > base，且仍同日
candidate := anchor
for {
  candidate = candidate.Add(interval)
  if candidate.After(base) {
    if sameLocalDay(candidate, anchor) { return candidate }
    break
  }
  // 防止无限：若已跨天则跳出
  if !sameLocalDay(candidate, anchor) { break }
}
return Date(nextDay..., h,m,0,0, ...) // 次日首档
```

或等价 `diff = base.Sub(anchor); steps = diff/interval +1; candidate = anchor + steps*interval` 判断同日。

需同步检查 `computeIntervalStartRun` 被 `normalizeInput` / `Update` 等多处调用（创建与编辑时），应统一。

---

## 四、影响与建议

- **影响**：所有 `interval` 触发器若 `start_time` 早于创建时间，均会 *首次等到次日首档*，用户感知为“为什么不是当天”（截图即此）。
- **数据**：`automation_rules NextRunAt` 已落库为次日，修复后需 `UPDATE` 或等待次日首跑后 `advance` 自行回到小时递进；现有 `scheduleOnce` 的 `advance` 已正确，后续小时递进无问题，**仅首次** 异常。
- **建议修复任务**：新建 `fix-interval-next-run`，改 `computeIntervalStartRunAt` 为同日递进，补 `service_test.go` 用例 `00:05 每1h at 14:00 → 14:05`，`go vet/type-check` 后发布 `0.0.5`。

---

## 附：取证

```bash
sed -n '154,177p' internal/automation/service_schedule.go
grep -n "computeNextRun\|NextRunAt" internal/automation/service.go | head
go test -run TestComputeNextRun -v 2>&1 | head
```
