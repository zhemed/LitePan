# 执行摘要（lightweight，联网资料核对）

- 目标：确认 Alist 侧如何避免"服务器取链 → 手机下载"导致的 115 签名失败。
- 做法：直接抓 Alist 上游仓库三个文件（115 驱动实现、存储模型、115 驱动配置）。
- 结论：Alist 的 115 驱动同样把客户端 UA 传给取链（`DownloadWithUA`）→ 直链同样绑 UA；
  代理开关在存储通用设置里：`web_proxy`（默认 false）与 `webdav_policy`（`302_redirect` 会踩坑）。
  给用户的建议：115 存储打开「Web 代理」，WebDAV 策略避开 `302_redirect`。
- 未改本项目代码/配置；未访问用户内网主机。
