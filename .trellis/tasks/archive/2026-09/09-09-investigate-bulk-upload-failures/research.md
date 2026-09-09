# 调查报告：2000 文件批次上传失败（2026-09-09 21:5x）

## 1. 现场快照（取证时点）

| 指标 | 值 |
|---|---|
| 批次 | 2000 任务（bulk_0000~1999，1MiB/个） |
| 状态 | 64 success / 2 failed / 5 running / 1929 pending（**失败不阻塞批次**） |
| 失败率 | 2/66 已尝试 ≈ 3% |
| 失败文件 | `bulk_0010.bin`（21:50:31）、`bulk_0029.bin`（21:51:03）——相隔 32s，都在 progress=0 阶段 |

## 2. 错误本体（两例相同）

```
DRIVER_ERROR: 天翼云盘 API HTTP 511:
{"code":"S3ClientException","msg":"sessionKey=cf170a85-…,Unable to execute HTTP request: Read timed out,requestId=…"}
```

**189 云盘自己内部的 S3 网关读超时**（HTTP 511 非标准状态码，189 网关自定义）——上游服务端瞬时故障，不是本方网络/凭据/文件问题（两文件与其余成功文件同为 1MiB 随机内容）。

## 3. 为什么没有自动重试

错误文本无"上传分片 x/y"前缀 → 死在 **init/commit 类步骤**（`uploadEncryptedRequest → rawJSON`），不是分片 PUT：

- 分片 PUT（`putUploadPart`）：3 次重试，且状态码分支认 `429 || >=500` 为可重试 ✓
- init/commit/getMultiUploadURLs：`retryableUploadURLFailure` **只认传输层错误**（`url.Error`/`net.Error`/EOF）——rawJSON 返回的"HTTP 511"是包装后的 domain 错误，不匹配 → 判不可重试 → **一次 511 直接判任务失败**
- 该分类器与上游 `origin/main` 逐字一致——**上游共享的局限，非本方移植引入**

时间窗重建：两例相隔 32s，含各自的重试延迟（`retryDelay`）——189 S3 网关抖动约 1 分钟，之后恢复正常（64 个成功跨该时段持续产生）。

## 4. 影响与处置

- 批次继续跑，1929 个不受影响；结束时对 2 个失败任务在 UI **重新上传**即可（Resume 支持 failed 状态）。
- 若 189 侧抖动频繁，失败率会随批次规模线性累积（2000 文件 × 3% ≈ 60 个需手动重试）。

## 5. 修复建议（可选，另建任务）

把上传路径（init / commit / getMultiUploadURLs）的 HTTP 5xx/429 纳入可重试分类：`rawJSON` 层把状态码带进错误（结构化字段），`retryableUploadURLFailure` 增加对它的识别，配合既有 `retryDelay` 重试 2-3 次。改动小（一个分类函数 + 错误构造），能显著降低大批量批次的失败残留。上游同款局限，属本方可独立改进点。

## 6. 验证记录

- `upload_tasks` 失败子集逐行核对（错误/阶段/时间戳）。
- 容器日志：仅两条"上传文件失败"WARN 与该批次对应，无其它异常。
- `upload.go` 重试链路（putUploadPart 3 次 / getMultiUploadURLs 3 次 / init-commit 无状态码重试）逐段核实。
- 与 `origin/main` 分类器逐字对照确认上游同款。
- 全程只读，工作区干净。
