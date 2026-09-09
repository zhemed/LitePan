# 调查报告：上游合并安全性（2026-09-09）

数据基线：本方 `main=b77c1f7`（0.0.12），上游 `origin/main=374affd`（tag `v0.5.4-beta`），merge-base `4c160d9`。上游新增 **20 提交 / 343 文件 / +8245 -1474**；本方自 merge-base 后 **178 提交**（大量功能删除）。

## 1. 结论：C — 不建议整体 merge

`git merge-tree --write-tree` 模拟合并产生 **68 个真实冲突文件**，其中 **50 个是"本方已删除 × 上游仍修改"的 delete/modify 冲突**（STRM/跨盘/整理/公告/AI/夸克TV/8个驱动等已删功能），18 个是本方仍保留文件的内容冲突。即使逐一手工解决（全部"保持删除"），上游代码假定完整功能矩阵，`router.go`/`wire_http.go`/`automation×5` 的冲突解决等于重做一遍精简工作，风险远大于收益。**建议放弃 merge，改为按需 cherry-pick 保留功能相关的小修复。**

## 2. 上游 20 提交分类

### 2.1 属于本方已删功能（不可移植，12 个）

| 提交 | 主题 | 对应本方删除面 |
|---|---|---|
| 9aa08fe | 跨盘传输功能扩展 | cross_transfer（已删） |
| 906b7cb | STRM清理元数据 | strm（已删） |
| b2ecd58 | fix115strm | strm + 115 transport |
| d08cd04 | emby补全媒体信息 | mediaorganize（已删） |
| 6b3b379 | 整理分类未命中放指定文件夹 | classifyorganize（已删） |
| 9e75c6b | 修复目录整理误匹配 | classifyorganize（已删） |
| a1162dc | 修复手动匹配 | mediaorganize（已删） |
| f90d890 | AI初始化不渲染 | aiorganize（已删） |
| 87f4b2d | 联动刷新→清理账号缓存 | 联动/webhook 面（已删） |
| dd4c13d | 光鸭本地扫码 | drivers/Guangya（已删） |
| b73e6c2 | 115open离线ed2k | offline_download（已删） |
| 374affd | 秒传星图手机端优化 | 秒传/星图生态（本方精简面） |

### 2.2 与保留功能相关（可评估移植，8 个）

| 提交 | 主题 | 触及本方保留文件？ | 移植难度 |
|---|---|---|---|
| 8e332f3 | 认证刷新优化（oauth 状态码细分：401/400/429/403 + UA 显式携带） | `internal/httpx/oauth.go`+18（本方未改→干净）、`internal/api/oauth.go`+3（本方仅±1→可自动合并）、新增 `oauth_test.go` | **低，最值得** |
| 1c71fec | 账号连接检测优化（ping 重构 +89/-16 + 新增 ping_test） | `internal/account/ping.go`（本方未改→干净）、`drivers/115_Open/driver.go`（本方改过 ±→小冲突）、Baidu 部分不适用 | 低 |
| c7a424c | 修复一系列认证问题（189Cloud auth/transport/auth_response 新文件 +115_Open/auth） | 189Cloud auth.go +20/-25、transport +16/-4、driver +6/-2（本方均未改→干净，但需连带新增 `auth_response.go` 126 行、`internal/account/service.go`+21） | 中（混入大量无关改动，需拆分） |
| 353b830 | 修复服务器上传目录错位 | `web/src/components/file/CloudLocalUploadPanel.vue` +108/-8（本方未改、文件仍在→干净） | 低 |
| de83b46 | 上传任务批次化/任务树分层 | `internal/upload/manager.go`+60/-37（本方删了89行→**冲突+语义相撞**）、`worker.go`（本方删了423行→**冲突**）、上传前端全套 | **高，不建议** |
| ce0992b | 优化后台响应（169 文件） | 混合面 | 高（需逐文件筛） |
| a9458b2 | 部分显示优化（172 文件） | 混合面 | 高（需逐文件筛） |
| 2eb66b4 | Releases v0.5.3 | 版本文件 | 不适用（本方 0.0.x 独立版本线） |

## 3. 冲突清单（68 个）

**delete/modify（50，解决方式一律"保持删除"）**：
- `internal/strm`×6、`strmscrape`×1、`crosstransfer`×1、`cacheretention`×1、`classifyorganize`×3、`mediaorganize`×4（含 planner/executor）、`announcement`×2 + `api/announcement`×2、`aiorganize`×2、`quarktv`×3、`coverextract`×1
- 驱动：`Guangya`×4、`OpenList`×3、`123_Open`×3、`OneDrive`×2、`Baidu_Open`×2、`139Cloud`×2、`Quark`×1、`115_Open/offline_download.go`×1
- 前端：`crossTransfer.ts`、`CrossDriveTransfer.vue`、`ClassificationToolCard.vue`、`OfflineDownloadModal.vue`、`api/cross_transfer_admin.go`

**content 冲突（18，本方仍存在）**：`README.md`、`internal/api/router.go`、`internal/api/web/index.html`、`internal/app/wire_http.go`、`internal/automation/{service,service_run,service_test,service_validate}.go`、`internal/domain/automation.go`、`internal/upload/{manager,manager_test,worker}.go`、`web/src/api/automation.ts`、`web/src/api/cloudTools.ts`、`web/src/components/admin/AutomationPanel.vue`、`DashboardManagement.vue`、`web/src/components/upload/TaskPanel.vue`、`web/src/views/AdminView.vue`

其中 automation 5 文件 + domain/automation.go 是本方 0.0.11/0.0.12 重写面（trigger 仅 daily/interval、interval_minutes），上游变更会把 webhook/联动逻辑带回来，属于方向性冲突，不可自动解决。

## 4. 建议方案

1. **不合并** origin/main；保持本方独立 0.0.x 版本线。
2. 如需上游修复，按价值排序逐个小步移植（每项单独建 Trellis 任务、单独验证）：
   - P1 `8e332f3` 认证刷新优化（干净，含测试）
   - P1 `1c71fec` 账号连接检测（去掉 Baidu 部分）
   - P2 `c7a424c` 中 189Cloud 认证修复子集（需连带 auth_response.go / account/service.go）
   - P2 `353b830` 上传目录错位（单文件前端）
   - P3 `de83b46` 上传批次化（与本方精简 worker 冲突大，仅当用户需要大批量上传 UX 时再评估）
3. 移植后统一 bump `0.0.13+` 走标准发布管线。

## 5. 验证记录

- `git merge-tree --write-tree --name-only HEAD origin/main` exit=1，输出 68 冲突文件（/tmp/mtree.txt）。
- 保留文件上下游改动量与本方改动量逐一对照（见 2.2 表）。
- `web/src/components/file/CloudLocalUploadPanel.vue` 在本方存在且自 merge-base 未改，`353b830` 可干净 apply。
- 本次调查全程只读（fetch/merge-tree/diff/show），工作区仍干净，未执行 merge/cherry-pick。
