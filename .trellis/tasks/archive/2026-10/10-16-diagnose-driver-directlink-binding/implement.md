# 执行摘要（lightweight，只读）

- 步骤：读 `drivers/all.go` 注册表 → 全仓搜 Alist → 读 `drivers/115_Open/ops.go`、`drivers/189Cloud/ops.go`、
  `drivers/LocalFs/download.go` 的 `ResolveDownload` → 汇总于 `research.md`。
- 结论：只有 115 把客户端 UA 带进取链（已修）；天翼不带（待实测）；LocalFs 强制代理（不存在该问题）；
  Alist 在本仓**已不存在**。两跳握手驱动无关，自动覆盖将来任何 302 驱动；IP 绑定/需 Referer 两类只能走代理。
- 未访问外部主机；未改代码/配置。
