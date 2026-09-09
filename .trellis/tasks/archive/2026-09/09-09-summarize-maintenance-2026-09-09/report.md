# 维护总结报告（2026-09-09）

> 范围：`0.0.12`（`b77c1f7`，间隔分钟化发布后）→ `0.0.15`（`e2c492a`）。
> 5 个 Trellis 任务全部走 `create → prd → start → implement → archive → add_session` 闭环，journal Session 67–71。

## 一、任务与发布总览

| # | 任务 | 交付 | 版本 |
|---|---|---|---|
| 1 | `09-09-investigate-upstream-merge-safety` | 上游合并安全性调查：`merge-tree` 模拟 68 冲突（50 delete/modify + 18 content），结论 **C 不整体合并**，给出 5 项可移植修复分级 | （纯调查） |
| 2 | `09-09-adapt-upstream-fixes` | 按精简后代码移植上游 4 组修复（8e332f3/1c71fec/c7a424c 子集/353b830）+ 驱动嵌入 AuthRefreshControl | **0.0.13** `sha256:67d3b57f` |
| 3 | `09-09-fix-189cloud-error` | 修复移植引入的 189Cloud HTTP 400 判定回归 + 7 例回归单测 | **0.0.14** `sha256:3b2ae4fb` |
| 4 | `09-09-record-admin-cred-verify-014` | admin/123456 授权重置+落档；发现 form 编码陷阱；0.0.14 实测验收闭环 | （运维） |
| 5 | `09-09-cleanup-orphan-db-tables` | 迁移 0022 清理 7 张孤儿表 + 2 个 STRM 残留键 + quarktv 死代码 | **0.0.15** `sha256:da66cbe8` |

当前态：`main=e2c492a` 已推 GitHub，tags `v0.0.13/14/15` + releases 齐全，容器 `latest`=0.0.15 运行健康，工作区干净。

## 二、上游合并策略（本轮核心决策）

- 上游 `Ponphil/LitePan` `4c160d9..374affd`（20 提交/343 文件）与本方 178 提交重度分叉：模拟合并 68 冲突，其中 50 个是"本方已删功能被上游修改"，automation 5 文件是方向性冲突（上游会把 webhook/联动带回来）。
- **决策：永不整体 merge，保持独立 0.0.x 版本线，按需小步移植**。移植原则：
  1. 本方未改动的文件取上游最终版（checkout），本方改过的手工合并；
  2. 明确剔除已删功能面（STRM/跨盘/整理/emby/AI/光鸭/115离线/上传批次化）；
  3. 依赖闭包逐文件核实（本轮追出 `domain/conn_error`、`driver/auth_control.go` 等必需传递依赖）。

## 三、移植内容与适配点（0.0.13）

- OAuth 代理状态码细分（401/400带token失败/429/403）+ 刷新/转发显式 UA
- 连接检测重构（网络重试 600ms、限流/权限/认证分类、友好文案+诊断详情），剔除 Baidu
- 189Cloud 认证响应解析重构（新 `auth_response.go`，不回显 token）、新 refresh_token 先落库再刷会话、会话错误降级
- 115 刷新接入统一分类；Init 移除内联 Ping（**保留本方 600s 客户端超时**——上游回 30s 会伤大文件传输）
- 适配点：`RecoverAccount` 保持本方 void 签名（不连带移植 internal/auth 重构）；guard 为 nil 时新代码自动退化为原直刷路径

## 四、回归事件与教训（0.0.13 → 0.0.14）

- **现象**：验收 189Cloud 报 `HTTP 400 UserInvalidOpenToken/unifyAccountInfo is null` 循环且不恢复；DB 认证态却 `active/无错误`，调度器按健康态排到 7440 分钟后。
- **根因**：上游 c7a424c 把会话失效判定收窄为 `401 || (200 && payload)`（上游有 internal/auth 守卫接线兜底，本方未移植）；189 现网对失效 token 实际返回 **HTTP 400** + 失效 payload → 判成 DRIVER_ERROR → 被动刷新/Init 重刷/调度恢复全链路断。
- **修复**：恢复 `400 + is189AuthExpiredPayload → CodeAuthExpired`（payload 精确匹配，普通 400 不误判；保留上游 403/429 分类）+ taskauth 异步事件 ctx 加固 + 7 例回归单测。
- **实测闭环**：UI 请求触发被动刷新 → 凭据落库（`last_refresh_at=20:16:30`）→ 随后列表直接成功。

**教训入档**：移植上游"收窄型"修复时，必须确认其兜底机制是否同样被移植；上游行为差异（400 vs 401）要在回归测试里锁死。新回归单测 `drivers/189Cloud/transport_test.go` 7 例全绿。

## 五、凭据与运维知识（Session 70）

- `admin/123456` 经用户授权重置落库（`pkg/security.HashPassword` 生成，自校验通过；重置前全库备份 `/tmp/litepan-backup-20260909-201827.db`）；AGENTS.md 已更新。
- **API 登录陷阱**：`POST /api/auth/login` 仅接受 **form 表单编码**；JSON 体被静默解析为空用户名（日志特征 `username=""`）。已写入 AGENTS.md。
- 澄清：本实例 DB 证实 08-30 后从未改密（用户的改密发生在别处）。

## 六、数据库卫生（0.0.15，migration 0022）

- 逐表对照代码引用分类：`strm×3`/`offline_download_tasks` 纯孤儿；`media_organize_tasks`/`cache_retention_tasks` 仅 backup.go 引用；`quarktv_bindings` 纯死代码。
- **复核保留** `notifications`（站内通知，6 路由全接线）/`api_keys`/`fuse_mounts`——与"残留"直觉相反，取证纠正。
- 迁移 0022 幂等 DROP ×7 + 删 configs 残留键 `strm_base_url`/`strm_token`；现有库与全新安装 schema 一致收敛（表 17→10、键 10→8）。
- 删除 quarktv 死代码 4 处（repo/domain 文件/Store 装配/通知常量）；backup.go 同步清引用。
- 部署前备份：`data/backups/manual-pre-0022-20260909-202635.db`；启动自动迁移（ledger 记 version 22）。

## 七、质量与验证状态

- `GOWORK=off go vet ./...` 全绿（每版发布前执行）。
- `go test`：本轮新增/触及包全绿；**存量失败** `internal/file` `name_align_test` 2 例（中文数字集号解析）——经 stash 在 clean HEAD 复现，与本轮改动无关，已备案（可另建任务修）。
- 每次发布实测：health → admin 登录 → 天翼云盘 files/list 真实目录返回。

## 八、遗留事项

1. `internal/file` 中文集号解析 2 例存量测试失败（独立小任务可修）。
2. 上游仍有未移植项（c7a424c 的 internal/auth 守卫接线、de83b46 上传批次化 P3）——按需另行评估。
3. 上游继续演进（当前已到 v0.5.4-beta+），后续若要新修复，沿用"调查→分级→小步移植→回归单测"流程。

## 九、数据速查

| 项 | 值 |
|---|---|
| 版本线 | 0.0.12 → 0.0.13 → 0.0.14 → 0.0.15 |
| 镜像 | 105MB，三 tag（0.0.x / v0.0.x / latest）同 digest |
| 提交 | 14 个（含 3 个版本提交、4 个归档、4 个 journal、1 测试、1 修复、1 chore） |
| DB | 10 表 / 8 键（清理前 17 表 / 10 键） |
| DB 备份 | 2 份（重置密码前 / 迁移 0022 前） |
| journal | Session 67–71（journal-2.md，450 行） |
