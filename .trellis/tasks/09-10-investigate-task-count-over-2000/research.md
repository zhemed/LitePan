# 调查报告：任务数突破 2000 + 上传优化覆盖面（2026-09-10 凌晨）

## 1. "突破 2000"的完整对账（DB 5816 条）

| 来源 | 条数 | 状态 |
|---|---|---|
| 旧批次（21:50，用户已删 3 条失败记录） | 1997 | 全部 success |
| 我的补传探针（23:52，p1921/p1987） | 2 | success |
| **23:58 用户第二次全量上传（第一波）** | **2000** | success 720、running 5、余排队——**正在跑** |
| **00:01 又触发一次（第二波）** | **1816** | **纯 pending（走路中途停止）——纯重复** |
| 新波唯一失败 | 1 | bulk_0491，00:06，189"服务暂时不可用" |

双重触发根因：面板无防重复触发（每次生成新 client_task_id，服务端按 client_task_id 去重拦不住）。第二波若跑完会把 2000 个文件覆盖重传一遍（overwrite，数据无害、白耗时）。**用户决策：不动，全部跑完后查终局。**

## 2. 上传优化覆盖面（对 115 的回答）

| 优化 | 层级 | 覆盖 115？ |
|---|---|---|
| 批次化/任务树/SSE 定向广播/虚拟滚动（0.0.18） | manager+前端，驱动无关 | ✅ 覆盖（115 是 LocalUploader） |
| 目录解析 walk（`ensureLocalUploadTargetDir`，api/local_upload.go:538） | files.List/CreateFolder 驱动无关 | ✅ 115 走同一逻辑（115 CreateFolder ops.go:259） |
| 目录缓存 TTL 10min + 批次预解析（0.0.19） | ⚠️ **接线错误**：预解析写入 manager.targetDirCache，但批次创建走 handler 私有 resolver（自带 per-walk 前缀缓存），**共享缓存无人消费** → 预解析实为无效代码 | 半覆盖（walk 前缀缓存有效，跨请求缓存/预热未生效）——189/115 同样未受益 |
| 瞬时错误重试（0.0.20/0.0.21） | 189 专属（uploadEncryptedRequest + retryableUploadURLFailure） | ❌ 不覆盖 115；115 自带 uploadConfirmAttempts（确认阶段）+ mapResponseError，init/commit 类 5xx 重试缺口待单独评估 |
| 间隔门 | 各驱动自带（189=500ms；115 transport.go:202 有自己的门） | ✅ 各自有效 |

**诚实更正**：0.0.19 的"批次目录预解析"接线到了无人消费的缓存——当晚扁平测试目录本就不触发目录解析，40/min 的提升来自节奏本身而非预解析。深度树场景的真实收益需把 walk 接到共享缓存（驱动无关，115 同益）——可作为后续任务。

## 3. 监视计划

后台轮询：两波（created_at > 23:50 窗口）无 pending/running 即视为跑完 → 拉终局统计（成功/失败/每分钟曲线/失败归因 0.0.21 重试是否吃掉瞬时错）。

## 4. 验证记录

- DB 对账（总数 5816、分波直方图 23:52/23:58×2000/00:01×1816、同名双 pending 实证）
- running 任务全部来自第一波（bulk_0718-0722）
- ensureUploadTargetDir 唯一调用方=warmTargetDirs（预解析与 walk 脱钩实证）
- ensureLocalUploadTargetDir 前缀缓存算法 + 115 upload/transport 重试面初查
- 全程只读
