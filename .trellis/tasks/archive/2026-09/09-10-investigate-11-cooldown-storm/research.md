# research.md — 10.0.0.11「大面积上传暂缓：账号网络冷却」取证记录

> 调查时间：2026-09-10 22:36–22:44 CST（授权：用户当轮显式提供 SSH/app 凭据并要求排查）
> 目标机：`10.0.0.11`（fnOS / Debian 12），LitePan Web `:5211`
> 原则：**全程只读**（SQLite 一律 `mode=ro`，仅 `SELECT`；容器未重启、未改配置、未改数据）

---

## 1. 生产机基线与版本对照（R3）

| 项目 | 值 | 证据 |
|---|---|---|
| 镜像 | `ghcr.io/zhemed/litepan:v0.0.31` | `docker ps` / `docker inspect` |
| ImageID | `sha256:575f751e5b63…a7738aa` | 与本地 `0.0.31`/`latest` 镜像 ID **完全一致** |
| 容器启动 | `2026-09-10T13:13:40Z`（21:13:40 CST），`RestartCount=0` | `docker inspect -f '{{.RestartCount}} {{.State.StartedAt}}'` |
| 宿主机 | `fnOS`，`Debian GNU/Linux 12 (bookworm)`，`/dev/sda2` 32G 用 27% | 只读 `cat /etc/os-release` / `df -h` |
| 账号 | 2 个：`1=天翼云盘`、`2=115网盘` | `GET /api/admin/accounts`（只读接口） |

**结论**：生产机跑的就是含 `0.0.23/0.0.24/0.0.30/0.0.31` 全部冷却相关修复的版本 → 本次现象**不是「旧版本没打补丁」**。

## 2. 现象量化（R4）

DB（`/vol1/1000/docker/litepan/data/litepan.db`，只读）+ 内存态（API）对照：

| 维度 | 值 |
|---|---|
| 任务总数 | **822**（`source_type=server_local` 822、`driver_type=115_open` 822、账号全部 `account_id=2`=115网盘） |
| DB 状态 | `paused 784` / `pending 19` / `success 19` / **`failed 0`** |
| 界面（内存）状态 | `paused 803` / `success 19`（**无 pending、无 failed**） |
| 任务创建时间 | 21:56:21 – 22:02:06（由自动化规则生成，见 §5） |
| 成功时间分布 | 21:56–21:58（16）、22:09（2）、22:29:30（1） |
| 唯一「暂缓」事件 | **22:30:17.913 – 22:30:18.264（0.35 秒）** |
| 暂缓条数 | **183 条** INFO，`account_id=2`，`retry_after_seconds=30`（183/183 同值），**183 个互不相同的文件名**（每任务恰好 1 条） |
| 恢复/提交污染 | 无：`failed=0`，任务进度/`resume_data` 未丢失，`pragma quick_check=ok` |
| 当前负载 | `docker stats` CPU 0.01%，队列空闲（批次处于暂停态） |

按秒分布：`22:30:17 → 19 条`、`22:30:18 → 164 条`。

## 3. 根因链（R5）

1. **账号级网络退避被触发**：`internal/core/driverexec/exec.go:28-31` `netFailThreshold=3` / `netBackoff=30s`；`Run()` 连续 3 次 `domain.IsNetworkError(err)` 后 `recordNetFail()` 置 `until=now+30s`。
2. **退避期内每次派发零 I/O 短路**：`Run()` 开头 `backoffRemaining()>0` → 直接返回 `CooldownError(30s)`（`exec.go:82-84`），**不访问上游**。
3. **上层把它当作「可等待」如实记录**：`file.Service.UploadLocal` 的 `isRetryableCooldown` 分支 → `s.log.Info("上传暂缓：账号网络冷却，稍后自动重试", retry_after_seconds=30)`（`internal/file/service.go:372-378`）。
4. **放大效应**：退避窗口内**每个被 worker 派发到的任务**都会走一次 1→3，于是「1 次冷却 × N 个排队任务 = N 条日志」。0.35s 内 183 条、183 个不同文件实证了这一放大（不是 183 次真实网络失败）。
5. **任务侧语义正确**：`worker.go:107-127` 对冷却错误不判失败（`failed=0` 佐证），`0.0.30` 的 requeue 保证任务不会成为孤儿。

**未能归因的部分（如实说明）**：触发这 3 次连续网络失败的上游调用**在任何日志中都不存在**——
- 容器 `docker logs` 自 21:10 启动至今共 207 行，其中 183 行为上述暂缓、24 行为启动/认证/配置/登录，**零 WARN、零 ERROR**；
- 应用文件日志 `data/log/2026-09-10.log` 全部 184 条 entry 的 `level` 均为 20（INFO），无 DEBUG/WARN；
- `account_auth_states`（115）`status=active`、`last_error` 空、`last_failure_kind` 空 → 不是认证/刷新失败；
- 因此本次「冷却触发源」**无法从现有证据归因**（置信：机制链高，触发源无证据）。根因即在 `recordNetFail` 的**静默性**（见 §6 缺陷 B）。

## 4. 状态的 DB/内存分叉（附带发现，有实证）

19 个任务的**最终态在两处不一致**：

| 位置 | status | message | updated_at |
|---|---|---|---|
| DB（`upload_tasks`） | `pending` | `账号网络冷却中，30 秒后自动重试` | `1789050617.9813` |
| 内存/界面（API） | `paused` | `上传已暂停` | `1789050617.9816`（+0.3ms） |

即：worker 的冷却 patch（置 `pending` + 承诺 30 秒后自动重试）与并发的 `pause()`（置 `paused`）在同一任务上竞态，内存最终为「已暂停」，DB 最终行为「冷却等待中」；**提示语承诺的 30 秒自动重试对这 19 个任务并未发生**。

**风险（可复现）**：`internal/upload/persist.go:80-91` 启动恢复时对 `status='pending'` 的行执行 `go m.runTask(id)` —— 一旦容器重启，这 19 个任务会**自行恢复上传**，与界面「已暂停」矛盾（无需用户点「继续」）。

## 5. 批次来源（背景）

`automation_rules` 规则 #2 「定时全局备份」：`daily`，`time=00:22`，动作 `local_upload`
`{"account_id":2,"conflict_policy":"overwrite","mappings":["我的文件","杂物间","pve_backup","pve_hermes","SanDisk_CZ880_1T"],"target_parent_id":"/"}`。
`automation_runs` #3：manual，21:56:01–22:02:06，`created=822 / scanned=822` → 正是本批 822 个任务。

**相关**：822 条任务 `batch_id` 全为空 → `internal/upload/breaker.go:61-65` 的「批次熔断安全网」要求 `BatchID` 非空，故**对本自动化批次不生效**（本次未触发失败判死，影响有限，但属可改进项）。

## 6. 缺陷清单与修复建议（供后续独立修复任务）

| # | 缺陷 | 位置 | 建议 |
|---|---|---|---|
| A | **日志放大**：1 次账号冷却 × N 任务 = N 条 INFO（0.35s 183 条），运维侧看起来像「大面积故障」 | `internal/file/service.go:372-378` | 账号级冷却窗口内只记 1 条汇总（含暂缓任务计数），其余降 Debug；或把「暂缓」改为任务态而非日志流 |
| B | **触发源静默（可观测性缺口）**：进入/退出 30s 退避无任何日志或事件，导致「为什么冷却」在日志里查不到（本次即无法归因） | `internal/core/driverexec/exec.go:129-158` | 触发退避时 Warn 一条（account_id、连续失败次数、最后一次错误原因、退避时长）；恢复时 Info 一条；冷却短路计数周期性汇总 |
| C | **冷却 patch 与 pause 竞态 → DB/内存分叉 + 重启自动续传** | `internal/upload/worker.go:107-127`、`lifecycle.go:41-76`、`persist.go:80-91` | 冷却 patch 需在锁内校验状态仍为 `pending`（不得覆盖同期 `paused`），`ctx.Done()` 分支需落库为终态；恢复逻辑对 `pending` 行增加「非用户暂停」校验 |
| D | **批次熔断对无 `batch_id` 的自动化批次不生效** | `internal/upload/breaker.go:61-65` | 自动化批次写入 `batch_id/batch_name`，或将熔断键退化为 `account_id+target` 口径 |

## 7. 处置结论（R6）

- **不需要紧急处置**：0 失败、任务与进度完好、队列空闲、账号认证有效、DB `quick_check=ok`；界面「已暂停」即当前真实状态，用户随时点「继续上传」即可恢复（`BatchResume` 路径正常）。
- **「大面积暂缓」不是大面积故障**：它是一次 30 秒账号级网络退避在 0.35 秒内被 183 个任务各自记录一次的放大结果；115 账号的网络退避本身是**保护机制**，30 秒后自愈。
- **建议修复**：A/B/C 三项（B 是本次无法归因的直接原因），D 为顺带项；建议另建实施任务（目标版本 `0.0.32`），最小复现：单测里用假驱动让 `exec.Run` 连续 3 次返回 `net.Error`，断言「退避触发有 1 条日志」「N 个任务只产生 1 条汇总」「冷却窗口内 pause 后 DB 与内存一致且重启不自动续传」。

## 8. 只读与保密合规（R1/R2 证据）

- 只执行：`docker ps/inspect/logs/stats`、`df`、`ls`、`cat /etc/os-release`、`uptime`、`python3` 只读 `SELECT`（`file:…?mode=ro`）。
- 未执行任何写操作：无 `UPDATE/INSERT/DELETE`、无 `docker restart/rm/run`、无配置修改；容器 `RestartCount=0`、`StartedAt` 保持 21:13:40 未变。
- 口令仅经 `sshpass -p` 内存传参，未写入任何文件（仓库/任务/脚本/历史均无）；凭据未出现在本记录。
- **唯一副作用**（如实披露）：22:37:38 以 `admin` 登录 `:5211` 做只读核对，产生 1 条应用日志（`管理员登录成功`，`ip=10.0.0.91`）与一个会话 cookie（本地 `/tmp/lp11.cookie`，已删除）。
