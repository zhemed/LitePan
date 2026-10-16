# Design：安卓两跳握手（保住 302 直链）

## 问题模型（实测）

115 CDN 直链签名 = f(取链 UA, 失效时间)，**与 IP 无关**。因此：
`客户端下载时的 UA == LitePan 取链时的 UA` → 200/206；不等 → 403 `invalid signature`。
安卓浏览器把附件**交接给系统下载组件**，组件 UA ≠ 浏览器 UA → 必然 403 → 手机报 `ERR_INVALID_RESPONSE`。

## 方案

**第一跳**（安卓浏览器导航）：**不 302**，返回一个"空附件"（`200` + `Content-Disposition: attachment` +
`Content-Type: application/octet-stream` + `Content-Length: 0`），于是浏览器把
**LitePan 自己的下载地址**（而不是 115 直链）交给系统下载组件。

**第二跳**（任意组件带着自己的 UA 回来请求同一地址）：LitePan 按**这次请求的 UA** 走原有 302 流程，
`Location` = 刚取到的 115 直链 → 组件用**同一个 UA** 去 CDN 取字节 → 签名匹配 → 下载成功。

**字节始终直连 115（服务器只重新取链 + 302），302 语义保留。**

## 关键决策

### KD1 触发条件（窄口径，避免误伤）

仅当**同时**满足：
1. `r.Method == GET`（HEAD 走原 302，供元数据探测）；
2. `ua` 像"安卓浏览器"（含 `Android` **且** 含浏览器标识 `Chrome/`、`Safari/`、`Firefox/`、`WebView`、
   `SamsungBrowser`、`MiuiBrowser`、`UCBrowser`、`Quark`、`HuaweiBrowser`、`HeyTapBrowser`、`EdgA/`）；
   —— 安卓系统下载组件的 UA（如 `AndroidDownloadManager/14 (Linux; U; Android 14)`）**不含**这些标识；
3. **`!intent.Inline`**：只对"下载"生效，**播放/内联不看不动**（否则安卓上看视频会被塞一个空附件）；
4. `PickAction(...) == ActionRedirect`（即账号确实是 302 模式；代理模式下 `ActionStream`，本逻辑不参与）；
5. 该资源在**握手窗口内首次**出现（见 KD2）。

### KD2 防循环：一次性窗口

`handoffTracker{ seen map[key]time.Time }`，`key = accountID + "\x00" + fileID`，窗口 **90s**：
- 首次 → 发空附件；
- 窗口内再次（无论 UA 相同或不同）→ 一律 302。

这样两种现实都覆盖：**浏览器自己回来取**（UA 相同：第二跳按浏览器 UA 取链 → 浏览器 UA 取 CDN ✓）与
**系统组件来取**（UA 不同：第二跳按组件 UA 取链 ✓），且永不循环。
窗口过期后再点下载 → 重新走握手的第 1 跳（用户重试体验正常）。

### KD3 与解析缓存的关系

`resolveCacheVariant(ua, playback)` 用**完整 UA** 作为缓存变体（`internal/playback/service.go:117`），
所以第一跳（浏览器 UA）与第二跳（组件 UA）取到的是**各自独立**的链，不会串味。
第一跳虽然会白解析一次（浏览器 UA 的链用不上），但换取"代理模式/播放路径零影响"，值得。

### KD4 空附件的文件名

`Content-Disposition: attachment`；若 `intent.FileName` 非空则附 `filename*=UTF-8''<percent-encoded>`。
空附件只是"交接凭证"，真实文件名由第二跳 CDN 响应给出，故缺名不影响结果。

### KD5 可测性

`handoffTracker` 暴露 `now func() time.Time` 供测试注入时钟（验证窗口过期后可再次握手）。
`Service` 在 `NewService` 中初始化 tracker；访问器对 nil 做懒初始化，兼容直接构造 `Service{}` 的既有测试。

## 边界与风险

| 风险 | 说明 / 处置 |
|---|---|
| 安卓"空附件"可能被当成已完成下载（0 字节）而不发起第二跳 | **只能真机验证**；若失败，回退提交（无数据/配置副作用） |
| 安卓系统组件是否携带 cookie | 第一跳与第二跳都走现有 `requirePublicOrAdmin` 鉴权；若组件不带 cookie 会 401 → 届时再评估短期令牌（本轮不做，避免放大改动面） |
| 播放/内联被误伤 | KD1 第 3 条；单测覆盖 `intent.Inline=true` 仍直接 302 |
| 代理模式被误伤 | KD1 第 4 条；单测覆盖 `ActionStream` 分支不受影响 |
| iOS / 桌面 | KD1 第 2 条；单测覆盖二者首次即 302 |

## Rollback

回退本次提交（纯 Go 逻辑，无迁移、无配置、无 embed 变化）；生产需重新部署上一版镜像。

## File Map

| 路径 | 动作 |
|---|---|
| `internal/playback/handoff.go` | 新增：UA 判定、一次性窗口、空附件写出 |
| `internal/playback/service.go` | 改：`NewService` 初始化 tracker；`ServeHTTP` 的 `ActionRedirect` 分支前插入握手判断 |
| `internal/playback/handoff_test.go` | 新增：单测（含反向验证） |
| `.trellis/tasks/10-16-fix-android-302-handoff/**` | 任务记录 |

**零改动（越界即失败）**：`web/src/**`、`internal/api/web/**`、`internal/api/**`（除非必要）、
`drivers/**`、`internal/store/**`、`go.mod`、`Dockerfile`、compose 文件、数据库。
