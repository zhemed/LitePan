# PRD：302 直链在手机端可用（安卓两跳握手）

## Goal

让 **302 直链模式**（`download_mode=redirect`）在安卓手机上也能下载成功，且**文件字节仍由手机直连 115 CDN**
（服务器只负责"重新取链 + 302"，不做字节中转）。电脑端与 iOS 的现有行为保持不变。

用户原话（2026-10-16）：**"你自己想办法，必须走302，先去调查方案"** → 方案 A 已获批准。

## Background（实测证据，来自生产 NAS 的 LitePan 实例）

1. **115 直链签名只绑定"取链时的 User-Agent"，不绑 IP**：
   - 同 UA、**跨机器（不同出口 IP）**下载 → `206` ✓
   - 换 UA（桌面 Chrome / 安卓 Chrome / 安卓下载器 / 无 UA）→ `403 {"message":"invalid signature"}` ✗
2. **手机失败的机制**：手机浏览器点下载 → LitePan 用**浏览器 UA** 经 `r.UserAgent()` 取链 → 302 到 CDN →
   安卓浏览器**不自己下载**，把**最终 URL（CDN 直链）**交给系统下载组件 → 组件用**另一个 UA** 取 CDN →
   `403`；该 403 的响应头是 `Content-Type: application/zip` + `Content-Disposition: attachment`（一个 85 字节的
   JSON 错误体）→ 安卓浏览器显示 `net::ERR_INVALID_RESPONSE`。
3. 电脑端成功的原因：桌面浏览器自己就是下载器，UA 全程一致。
4. LitePan 侧看不到任何失败记录（日志无 download/403 相关行）——失败发生在手机↔115 之间。

## Requirements

### R1 两跳握手（仅影响"会交接给系统下载器"的客户端）

- 第 1 跳（安卓浏览器导航到 `/api/files/download` 或播放重定向端点）：
  **不取链、不 302**，而是返回 `200` + `Content-Disposition: attachment; filename*=UTF-8''<名>`
  + `Content-Type: application/octet-stream` + `Content-Length: 0`，**响应体为空**。
- 第 2 跳（**任意**客户端带着自己的 UA 再次请求同一地址，含安卓系统下载组件）：
  按其**本次请求的 UA** 取链并 `302` 到 CDN → 该客户端随后用**同一个 UA** 取 CDN → 签名匹配 → 下载成功。
- **字节不经过服务器**：第 2 跳仍是 302，客户端直连 115 CDN。

### R2 不影响既有客户端

- 桌面浏览器、iOS Safari、`curl`/下载器等：**行为与现在完全一致**（直接 302，不经过空附件）。
- `force_proxy=1`（中转）与 `redirect=0`（返回 JSON URL）两条路径**逻辑不变**。
- 播放（非下载）路径的 302 行为不变。

### R3 防循环

- 同一资源在握手窗口内**只发一次空附件**：第 2 次及以后的请求一律 302（无论 UA 是否与第 1 次相同）。
  这样才能覆盖"浏览器自己回来取（UA 相同）"与"系统组件来取（UA 不同）"两种情况，且不会死循环。

### R4 可测

- 单测覆盖：安卓浏览器 UA 首次 → 200 空附件且无 `Location`；同资源第二次（同 UA）→ 302；
  第二次（安卓下载器 UA）→ 302 且 `Location` 指向 CDN；桌面 UA / iOS UA / HEAD → 直接 302。

## Constraints

- 只改后端（`internal/playback/**`，必要时 `internal/api/**`），**不动** 115 驱动取链逻辑、不改前端、不改数据库、
  不改部署形态；`internal/api/web/**`（embed）保持不变。
- 不得引入"字节中转"（除已有的 `force_proxy` 语义外）。
- 行为开关**不新增配置项**：仅按"安卓浏览器 UA + 302 模式 + 非 HEAD"判定，便于回滚（回滚=回退提交）。
- 本轮**不发版**；是否发版/部署由用户另行决定（依赖：更新到生产 NAS 需要新镜像）。

## Acceptance Criteria

- [x] 安卓浏览器 UA 首次请求 → `200`、`Content-Disposition: attachment`、`Content-Length: 0`、无 `Location`
- [x] 同资源第二次请求（任意 UA，含安卓下载器 UA）→ `302` + `Location` 指向 115 CDN
- [x] 桌面 UA / iOS UA / `HEAD` 首次请求 → 直接 `302`（与改动前一致）
- [x] `force_proxy=1` 与 `redirect=0` 行为不变（有测试断言）
- [x] 防循环：窗口内连续 3 次请求 → 只有第 1 次是空附件，其余 3 次中的后两次为 302
- [x] 质量门：`make lint` 0 issues、`go vet ./...`、`go test ./...` 全绿、`cd web && npm run type-check` 通过；
      前端与 embed 零改动
- [x] 单测中含"反向验证"：把握手开关去掉后相应用例失败（证明测试有效）
- [x] 未发版、未部署；`:5211` 与生产 NAS 未被改动（生产部署另行授权）
- [x] 任务归档 + journal

## Notes

- 交付后仍需**真机验证**（用户手机）——浏览器是否会把"空附件"当成已完成下载，只能实测；
  若实测失败，回退提交即可（无数据/配置副作用）。
- Scope 预判：`feature`（后端行为改动 + 测试）。
