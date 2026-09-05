# 验证0.0.14认证与0.0.15文件夹上传链 · 验证报告

验证时间：2026-09-05 · 方式：用户提供的管理员口令登录 `127.0.0.1:5211` API（口令仅内存变量使用，未写入任何文件/commit，本报告亦不含口令）

## 结论

**两项全部验证通过，另发现并修复 1 个 A-road 落库缺口（已发 `0.0.16`）。** 测试痕迹已清理（云盘测试目录按应用语义移入回收站，未清空回收站以保护用户数据）。

## 0.0.14 认证验证 ✅

- 登录成功（admin）；账号仅 1 个：天翼云盘（189，前提 active）；无 115 账号（该项无客观测条件）。
- 手动 `POST accounts/1/refresh-auth`：上游 189 回 429（token 有效期至 09-08，刷新过频被限）→ **新分类生效**：`RATE_LIMITED/retryable` → 状态 `cooldown` + 59s 后自动重试；**无 fatal、无"重新扫码"、未读通知 0**。正是本次合入要修的误报场景，反向实证通过。
- 数据通路：cooldown 期间列取云盘根目录正常（4 目录），会话有效、仅刷新被节流，符合设计。

## 0.0.15 上传链验证 ✅（含缺口修复）

- 面板 `browse` 两级目录正确；文件夹提交 `accepted:true count:3`（服务端展开，`buildLocalUploadSources` 新代码）。
- 首轮发现缺口：`batch_id/batch_name` 落库为空（`rel_path` 正常）→ 根因为 A/B 拆分时 `newTaskStateLocked` 的 batch 拷贝行落在 B-road 的 `manager.go` hunk 里。已手工移植 12 行（跳过 B-road broadcast 部分），发 **`0.0.16`**（`60c7882`，latest 已换，`github` 已推）。
- 复测 `verify-batch-003`：3 行 `batch_id/batch_name/rel_path/rel_dir` 全对（`batch_name` 由空 display_name 自动取目录名），状态 success。
- 云盘结构与源完全一致：`/__verify_0_0_15__/root.txt`（11B）、`sub1/a.txt、b.txt`（各8B）。

## 清理确认

- 测试任务 3 条已 `batch-delete`（DB 0 残留）；云盘测试目录已删入回收站（未清空回收站）；`local-upload` 配置恢复原始（disabled/空映射）；容器/宿主测试目录、cookie、临时补丁全删。

## 遗留说明

1. 115 无账号，本次未覆盖 115 驱动刷新；有号后可重测（同一接口）。
2. `CreateServerLocalTasks` 的 batch 透传（离线中转流）未合入，本地上传流不用它；若以后合入 B-road 需带上。
3. 全量 `go test` 唯一失败仍是基线已知的 `internal/file` 中文数字 2 用例。
