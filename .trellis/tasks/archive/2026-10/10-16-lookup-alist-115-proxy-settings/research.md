# 联网核对结果：Alist 的 115 直链/代理设置

来源（外部内容，按数据对待；均为 Alist 上游仓库原文）：
- 驱动实现：https://github.com/AlistGo/alist/blob/main/drivers/115/driver.go
- 存储通用代理设置：https://github.com/AlistGo/alist/blob/main/internal/model/storage.go
- 115 驱动配置项：https://github.com/AlistGo/alist/blob/main/drivers/115/meta.go

## §1 事实：Alist 与 LitePan 犯的是同一个设计

`drivers/115/driver.go` 的 `Link()`：

```go
userAgent := ""
if args.Header != nil {
    userAgent = args.Header.Get("User-Agent")
}
downloadInfo, err := d.client.DownloadWithUA(file.(*FileObj).PickCode, userAgent)
```

→ **Alist 也把"客户端的 User-Agent"传给了 115 的取链接口**（`DownloadWithUA`），
与我们 LitePan 的 `r.UserAgent()` 做法完全同型。因此 **Alist 拿到的 115 直链同样绑定这个 UA**；
只要后续真正的下载方换成别人（安卓系统下载组件、内置浏览器、迅雷等），就会 `403 invalid signature`
—— 手机端同样表现为下载失败 / `ERR_INVALID_RESPONSE`。

同文件还有一条佐证 115 校验很严的注释（原文）：

> The 115 CDN requires the account Cookie on the download request, but the SDK sends cookies at the
> transport layer so they are absent from downloadInfo.Header. Without it, proxied downloads fail with
> 403 "no cookie value". Restore the Cookie so proxy mode works.

→ 说明**"直连"与"由服务端取"是两种不同的信任模型**：直连靠签名（绑 UA），服务端取还要带上账号 Cookie。

## §2 开关在哪（确切字段名与取值）

115 驱动自身的配置项（`meta.go` 的 `Addition`）**没有**代理开关，只有一个 `limit_rate` 之类的业务项；
代理开关在**存储的通用设置**里（`internal/model/storage.go` 的 `Proxy` 组）：

| 字段（JSON） | 界面含义 | 取值 | 影响 |
|---|---|---|---|
| `web_proxy` | **Web 代理** | `true` / `false`（**默认 false**） | `false` → Web 端 `/d/` 走**直链/302**（客户端去取上游，**手机端会踩 UA 绑定坑**）；`true` → 由 **Alist 自己取字节**再转发给客户端（安全） |
| `webdav_policy` | **WebDAV 策略** | `302_redirect` / `use_proxy_url` / 其它（=native proxy） | `302_redirect`：客户端直连上游（**会踩坑**）；`use_proxy_url`：走 `down_proxy_url` 指定的下载代理；其它值：Alist 本机代理（安全） |
| `proxy_range` | 代理 Range | `true`/`false` | 与中转时的分段下载有关 |
| `down_proxy_url` | 下载代理 URL | URL | `use_proxy_url` 策略下用它 |
| `down_proxy_sign` | 下载代理签名 | 默认 `true` | 给代理 URL 加签 |

（`Webdav302()` / `WebdavProxy()` / `WebdavNative()` 三个判定函数见源码，取值即上表的字符串。）

## §3 给用户的可操作结论

1. **Alist 上的 115 存储，想手机端稳：把该存储的「Web 代理」打开**（`web_proxy=true`）。
   网页端下载就变成 Alist 中转，手机不再直接碰 115 的签名链接。
2. **用 WebDAV 读 115 的客户端**：确认「WebDAV 策略」不是 `302_redirect`；默认（native proxy）或
   `use_proxy_url` 都不会踩 UA 绑定。
3. **如果你就是要直链**（省服务器带宽）：那和 LitePan 之前一样，手机端可能失败 —— 除非该客户端
   从取链到下载都是同一个 UA（比如桌面浏览器自己下载）。
4. 反过来说：Alist 若已经开着 Web 代理却还失败，那问题多半不在 UA 绑定，而是 115 的 **Cookie** 校验
   （源码注释里的 403 "no cookie value"）或账号风控 —— 与本次 115 手机问题不是同一类。

## §4 边界与声明

- 以上为**上游源码原文**（README 级事实），非我实测；Alist 的 115 直链绑定行为**未在其上实测过**
  （我方实测对象是 LitePan → 115 的同型链路：换 UA 必 403）。
- 版本差异可能有（字段名在 Alist 3.x 稳定，但界面文案随语言包变化）；以你所用版本的存储编辑页为准。
- 本轮**未访问用户内网任何主机**，只访问了 Alist 官方仓库文件。
