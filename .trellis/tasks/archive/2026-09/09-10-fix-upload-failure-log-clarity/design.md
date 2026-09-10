# 设计

## 后端（internal/file/service.go）

```go
if err != nil {
    switch {
    case errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled):
        s.log.Debug("上传文件已取消", ...)
    case driverexec.IsCooldownError(err) != (0, false):  // 语义：冷却可等待
        seconds, _ := driverexec.IsCooldownError(err)
        s.log.Info("上传暂缓：账号网络冷却，将在稍后自动重试",
            "account_id", accountID, "name", req.FileName, "retry_after_seconds", seconds)
    default:
        code := ""
        if ae, ok := domain.AsAppError(err); ok { code = string(ae.Code) }
        s.log.Warn("上传文件失败", "account_id", accountID, "name", req.FileName, "code", code, "err", err)
    }
    return nil, err
}
```
导入：`litepan/internal/core/driverexec`（file 包已依赖该包，无环）。

## 前端（TaskPanel + 新纯函数模块）

1. `web/src/composables/upload/uploadFailureSummary.ts`
```ts
export function classifyUploadFailure(error: string, code?: string): string  // 网络异常/账号冷却/认证失效/限流/权限不足/存储冲突/其他
export function summarizeUploadFailures(tasks: UploadTask[], limit = 3): { total: number; parts: string[] }
```
2. TaskPanel：
   - 行状态：`message` 含「冷却」→ 状态列显示「重试中」（statusClass 仍为 pending 视觉）
   - 列表上方（或面板底部）在 `failed > 0` 时显示「失败 N · <原因> ×k ...」
3. 校验脚本扩展：`classifyUploadFailure` 与 `summarizeUploadFailures` 的断言（含中文原因、未知错误归"其他"、排序取前三）
