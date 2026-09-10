# 10.0.0.11 调查记录（2026-09-10 20:56）

## 一、主机与实例现状

| 项 | 值 |
|---|---|
| 主机 | **fnOS（飞牛 NAS）**，Debian 12，内核 6.18.18-trim，uptime 11h，盘 32G（用 27%），内存 3.9G |
| 实例 | 容器 `litepan`，镜像 **`ghcr.io/zhemed/litepan:v0.0.31`**（20:42 启动，即我 20:33 发布的版本） |
| 运行参数 | `Privileged=true`、`PidMode=host`、`Restart=always`，数据卷 `/vol1/1000/docker/litepan/data -> /app/data` |
| 只读挂载 | `/vol1/1000/docker`、`/vol1/1000/我的文件`、`/vol2/1000/杂物间`、`/vol3/1000/pve_backup`、`pve_hermes`、`SanDisk_CZ880_1T`（均 ro） |
| 服务健康 | `/api/health` ok；admin 登录成功；账号 2 个（id=1 天翼云盘、id=2 **115网盘**，均 active） |
| 任务 | **72 条：71 成功 / 1 失败**（当前空闲，成功数 12 秒无变化） |

## 二、那 1 个失败的根因（真实产品缺陷，跨驱动）

```
file_name: litepan.db-wal        account: 115网盘(115_open)
target: docker/litepan/data      progress 100%
total_bytes=3341352  uploaded_bytes=3374312   ← 上传途中文件变大了 32,960 B
error: DRIVER_ERROR: 网盘服务异常:
  Put "https://fhnfile.oss-cn-shenzhen.aliyuncs.com/...":
  net/http: HTTP/1.x transport connection broken:
  http: ContentLength=3341352 with Body length 3374312
```

**机制**：上传计划阶段 `StatLocalFile` 记录文件大小（3,341,352 B），随后驱动按该大小声明
`Content-Length` 并流式发送；而这期间 **SQLite 正在写自己的 WAL**，文件实际增长到
3,374,312 B → HTTP 客户端发出比声明更多的字节 → Go transport 判定协议错误并中断请求。

**性质**：任何"上传过程中会被修改的文件"都会触发（WAL/SHM、活跃日志、正在写入的备份）；
本例是用户把 LitePan 自己的数据目录当备份源 → 必然踩中。189 驱动因用 `io.LimitReader`
按计划大小截断读取，不会触发该错误；**115 驱动（OSS 直传）会硬失败**。

**日志佐证**：该实例日志内「上传文件失败」共 6 条，均为此类（WAL/其它变动文件 +
一次冷却「上传暂缓」INFO，说明 0.0.31 的新日志口径已生效）。

## 三、安全态势

| 项 | 现状 | 风险 |
|---|---|---|
| 5211 暴露 | `LISTEN *:5211`（所有网卡） | 同网段任意主机可访问；若 router 有端口映射则公网可达 |
| UPnP | `upnp_service` 正在监听 `10.0.0.11:49152` | NAS 可自行开端口映射 → 需在路由器核对 |
| 容器权限 | privileged + pid host + 多个宿主目录只读挂载 | 一旦实例被登录，宿主受影响面大 |
| SSH | `PermitRootLogin yes`（密码登录可用） | 与本机 10.0.0.91 同类风险 |
| 登录记录 | **0 条失败登录**；成功登录仅来自 `10.0.0.91`（本次排查） | 未见异常访问 |
| cron | 仅发行版自带（e2scrub_all/sysstat） | 正常 |
| 进程 | 无异常进程（系统 + trim 套件 + litepan） | 正常 |
| API 凭据暴露 | `/api/admin/accounts` 返回含 `access_token`/`refresh_token` 的 config | 仅管理员可读，但界面/日志若不慎截图即泄露 |

## 四、建议（待授权）

1. **产品修复（建议尽快）**：上传"变动中文件"的防御——
   a. 所有驱动发送体统一按计划大小 `LimitReader` 截断（115 驱动当前会发送实际长度）；
   b. 上传完成后比对文件当前大小与计划大小，不一致时给出**明确原因**（"文件在上传过程中被修改"）而不是底层 transport 报错；
   c. 前端失败归类新增该类（可在 0.0.31 的 `uploadFailureSummary` 里加"文件被修改"规则）；
   d. 文档/UI 提示：备份 SQLite 数据目录时应排除 `*.db-wal`/`*.db-shm`。
2. **运维加固（10.0.0.11）**：5211 改绑内网/关闭 UPnP 自动映射；SSH 关闭 root 密码登录改密钥；容器尽量去掉 privileged（评估 FUSE 需求）。
3. 该实例已运行 0.0.31，本次修复合并后可直接升级。
