# 候选清单 P7 / P8 / P9 关闭记录（零代码改动）

> 用户 2026-09-12 指示：**7、8、9 归档**。本文给出逐项状态核实与"为何可先不做 / 何时值得重启"，作为候选清单的收尾。

## 0. 结论

| 项 | 上游依据 | 处置 | 一句话理由 |
|---|---|---|---|
| **P7** | `ae75882` 子集（上传取消语义 / 批次名收敛 / resume 定时器） | **归档不实施** | 取消判定本方已内联等价；批次名显示是本方有意行为；定时器写法属代码卫生 |
| **P8** | `adca0ee`（铃铛通知改服务端 SSE 推送） | **归档不实施** | 功能项，非缺陷；当前轮询无问题，改动面涉及新端点 + 前端 |
| **P9** | `46a0a89`（播放诊断及直读文案） | **归档不实施** | 诊断属"出问题时才需要"的能力，当前无播放故障驱动 |

---

## 1. P7：上传取消语义 / 批次名 / resume 定时器

**（1）取消语义（`api/upload.go` 的 `isClientGone` / `isCancelError`）——本方已内联等价，无需移植。**

上游把三处内联判断抽成两个 helper：

```go
func isClientGone(r *http.Request) bool { return r != nil && errors.Is(r.Context().Err(), context.Canceled) }
func isCancelError(err error) bool { return err != nil && strings.Contains(strings.ToLower(err.Error()), "context canceled") }
```

本方同一文件已经是等价内联写法（三处）：

```
internal/api/upload.go:34   if errors.Is(r.Context().Err(), context.Canceled) || strings.Contains(strings.ToLower(err.Error()), "context canceled") {
internal/api/upload.go:64   （同上）
internal/api/upload.go:102  （同上）
```

⇒ 移植收益仅为"抽 helper"，**零行为差异**，不做。

**（2）批次名收敛（上游删除 `local_upload` 的 `batchName` 派生）——本方属有意行为，反向操作。**

```
internal/api/local_upload.go:288  batchName := strings.TrimSpace(in.DisplayName)
internal/api/local_upload.go:289  if batchName == "" && len(in.Items) == 1 && in.Items[0].IsDir {
internal/api/local_upload.go:290      batchName = path.Base(strings.Trim(cleanRelativePath(in.Items[0].RelPath), "/"))
```

上游在 `ae75882` 中移除了这段（其批次命名策略另作调整）。本方的批次行**需要**这个显示名（文件夹上传显示为单条批次），且 `0.0.36` 已按用户决策取消"自动化任务写批次身份"——两者不冲突。照搬上游删除会**改变本方面板显示**，故不做。

**（3）`upload/resume.go` 定时器写法**：上游把 `if timer != nil { timer.Stop() }` 简化为 `timer.Stop()`（防御式代码清理），属代码卫生、无行为差异。不做（且死代码清理任务已确认其中无孤儿子句）。

**重启条件**：出现"取消/中断后仍记 ERROR、任务状态异常、或上传中断处理有实际缺陷"时，再按上游差异逐条比对。

---

## 2. P8：铃铛通知改服务端实时推送（SSE）

- 上游 `adca0ee`：新增通知流式端点（`internal/api/notifications_stream_test.go` 等 4 文件）＋前端改为服务端推送，替代定时轮询。
- 本方现状：通知功能在线（路由 `/api/admin/notifications` 系列 + 前端铃铛），采用轮询；**无缺陷、无性能问题报告**。
- 属性：这是**功能增强**（实时性 + 少轮询），不是修复；改动面＝后端新端点 + 前端订阅 + 断线重连策略。
- 用户决定：**归档不实施**。
- **重启条件**：希望通知"秒级到达"、或想减少轮询请求（例如移动端省电/流量）时再开任务。

---

## 3. P9：播放诊断及直读文案

- 上游 `46a0a89`：`internal/playback/service.go` 诊断日志 + `strm_play.go` 直读文案（上游另有 STRM 相关改动，本方不适用）。
- 本方现状：`internal/playback/service.go`（192 行）在内，**无诊断日志**（`grep 诊断` → 0）；播放链路当前无故障报告。
- 属性：诊断能力属"出问题时才需要"的排障手段，无问题时留存价值有限（且会引入日志量）。
- 用户决定：**归档不实施**。
- **重启条件**：出现播放/直链（302）故障需要定位时，按上游差异移植并可同时补最小复现用例。

---

## 4. 本轮动作与候选清单最终状态

- 零代码改动（`git status` 仅本任务目录）。
- **候选清单（上游调查 §6）最终状态**：

| 项 | 内容 | 状态 |
|---|---|---|
| P1 | 上传超时修复 | ✅ 已移植（0.0.38 + 0.0.39 超时统一） |
| P2 | 客户端断连不记 ERROR | ✅ 已移植（0.0.43） |
| P3 | 认证收口 | 📦 归档不实施（用户决定） |
| P4 | 账号级目录缓存失效 | 🚫 不适用（目标动作已删） |
| P5 | 慢后台日志 + accountprofile | ✅ 已移植（0.0.43） |
| P6 | 目录整理 TMDB 误匹配 | ❌ 作废（模块为死代码，已删） |
| P7 | 上传取消语义/批次名 | 📦 归档不实施（本文 §1） |
| P8 | 铃铛 SSE 推送 | 📦 归档不实施（本文 §2） |
| P9 | 播放诊断 | 📦 归档不实施（本文 §3） |

⇒ **候选清单全部关闭**：**3 项已落地**（P1 上传超时 0.0.38 + 0.0.39 超时统一、P2 断连日志 0.0.43、P5 慢后台日志与刷新串行化 0.0.43）、**2 项作废/不适用**（P6 死代码作废、P4 目标动作已删不适用）、**4 项按用户决定归档**（P3 认证收口、P7 取消语义/批次名、P8 铃铛 SSE、P9 播放诊断）。
