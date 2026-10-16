# PRD：排查 115 直链在手机端打不开（net::ERR_INVALID_RESPONSE）

## Goal

用户反馈：**同一条 115 CDN 直链**，电脑端浏览器可以正常下载，手机端浏览器打开就报
`net::ERR_INVALID_RESPONSE`。要求分析出问题出在哪一环，并给出结论（若根因在 LitePan 侧，则另议修复）。

## Background（用户提供的事实）

- 现象：手机端打开该直链 → 浏览器错误页 `net::ERR_INVALID_RESPONSE`；电脑端同一链接可下载。
- 链接形态：`https://cdnfhnfile.115cdn.net/...`，带签名参数（`t=`(过期时间戳)/`u=`/`s=`/`d=`/`k=` 等）——
  属**有时效的签名直链**，由 115 下发。
- 本方（LitePan）相关路径：`/api/files/download` 走驱动 `ResolveDownload` → 302 直读（redirect）或
  本地代理（proxy）两种模式；115 驱动的直链由 115 开放平台返回。
- **本任务只做排查**：不下载该文件内容（最多做 HEAD / 1 字节 Range），不改代码；
  若结论指向 LitePan 缺陷，另开任务修复（需用户确认）。

## Requirements

### R1 复现与对照（最小网络动作）

- 对该直链只做**极小请求**：`HEAD`（或 `GET` + `Range: bytes=0-0`），**不下载完整文件**。
- 对照三组请求的**响应状态行 + 响应头**：
  1. 桌面 Chrome UA；
  2. 手机 Chrome UA（`Android ... Mobile Safari`）；
  3. 无 UA / 其它（用于区分"UA 决定行为"）。
- 记录：HTTP 版本、状态码、`Content-Type`、`Content-Length`/`Content-Range`、`Content-Disposition`、
  `Location`（若有跳转）、`Transfer-Encoding`、以及是否有非法/非 ASCII 头部值。

### R2 判定根因

按证据给出以下之一（或明确组合）：

- **115 CDN 侧**：如按 UA 分流（返回 `intent://`/app 唤起、或对移动 UA 返回非法响应）、签名/时效限制；
- **浏览器侧**：如 `Content-Disposition` 文件名非 ASCII 导致移动端解析失败、协议/头不合规；
- **LitePan 侧**：如本方的 302 直读丢失必要请求头、或代理模式下的响应头构造问题；
- **链接本身**：如已过期（`t=` 时间戳）或一次性。

### R3 可操作结论

- 明确"手机端为什么失败"以及**用户可用的绕行办法**（例如改用 LitePan 代理下载/换 UA/用 115 官方客户端）。
- 若根因在 LitePan，给出**最小修复方向**（文件 + 行为），但本轮不实施。

## Constraints

- **网络动作仅限**：对用户提供的那一条 115 CDN 直链做 HEAD / 1 字节 Range 对照，
  **不访问其它主机**、不下载大文件、不做扫描；不把该链接（含签名参数）写入仓库、任务记录或 journal。
- **只读**：不改代码、不改配置、不部署；不改归档任务与历史 journal。
- 凭据纪律：链接中的签名参数视为凭据，报告中不复述原文（用 `<直链>` 指代）。

## Acceptance Criteria

- [x] 完成三组 UA 的最小请求对照，记录状态码与关键响应头
- [x] 给出根因判定（CDN / 浏览器 / LitePan / 链接时效 之一或组合），并列出支撑证据
- [x] 给出手机端可用的绕行办法；若指向 LitePan，给出最小修复方向（不实施）
- [x] 零越界：`git status` 仅任务目录；未改代码/配置/部署；未把签名链接写入任何文件
- [x] 任务归档 + journal

## Notes

- Scope 预判 `lightweight`（只读排查，PRD-only）。
- 若用户要求把结论落成 LitePan 的修复（例如下载默认走代理、或补齐响应头），**另开任务**并在其中走质量门。
