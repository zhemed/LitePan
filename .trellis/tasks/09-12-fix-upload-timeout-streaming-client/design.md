# design.md — 09-12-fix-upload-timeout-streaming-client

## 1. 边界（change boundary）

| 层 | 文件 | 是否改 | 说明 |
|---|---|---|---|
| 共享网络设施 | `internal/httpx/client.go` | ✅ 新增一个构造函数 | 不改 `NewClient` 既有语义（其它调用方零影响） |
| 共享网络设施测试 | `internal/httpx/client_test.go` | ✅ 新增 | 覆盖新函数语义 |
| 驱动 115 | `drivers/115_Open/driver.go` | ✅ 加字段/初始化/关闭 | 只新增 `uploadClient`，`client`（600s）不动 |
| 驱动 115 | `drivers/115_Open/upload.go` | ✅ 数据面改用新客户端 + 分片重试 | 单请求与分片 PUT 两处 `d.client.Do` → `d.ossUploadHTTPClient().Do` |
| 驱动 115 测试 | `drivers/115_Open/upload_retry_test.go` | ✅ 新增 | 分类表测 + 假 transport 行为测 |
| 驱动 189 | `drivers/189Cloud/driver.go` | ✅ 上传客户端构造方式 | `uploadClient` 由 300s+关KeepAlive → streaming |
| 发版 | `README.md`、`docker-compose.yml` | ✅ 版本号 → `0.0.38` | 与历次发布一致 |
| 前端 | `web/**` | ❌ 不动源码 | 仅执行既有构建命令作为质量门 |
| 生产机 | `10.0.0.11` | ❌ 不触碰 | 无授权 |

**不做的事**：不改 115 分片大小策略（512MB 固定，用户定制）、不改 115 API 客户端 600s、不改 189 分片大小/重试分类/账号 500ms 节流、不改 `upload_task_concurrency`、不移植上游六个已删驱动的同类改动、不动 115 单请求上传的重试语义（上游亦未加）。

**行为缺口（最小可验证问题）**：数据面 PUT 受 `http.Client.Timeout` 总时长限制（115=600s、189=300s），大分片/慢链路会在传输途中被客户端掐断；且这一限制无法通过调整「API 客户端超时」修复（两者共用同一客户端）。行为**应**发生在数据面客户端的构造处（`internal/httpx` + 驱动 `Init`），而不是在调用点用重试掩盖。

## 2. 契约

```go
// internal/httpx/client.go（新增）
// NewStreamingClient 复用普通客户端的连接配置，但不限制整段文件传输时长。
func NewStreamingClient(base *http.Client, responseHeaderTimeout time.Duration) *http.Client
```

| 属性 | 约定 |
|---|---|
| `Timeout` | **0（不设）**——整段传输不受总时长限制 |
| `Transport` | `base.Transport` 为 `*http.Transport` 时克隆之（保留 Proxy / DisableCompression / IdleConnTimeout / MaxIdleConnsPerHost / DialContext keepalive）；否则克隆 `http.DefaultTransport` |
| `ResponseHeaderTimeout` | `>0` 时设置（本任务两处均 60s）；`<=0` 时不设置 |
| 关闭 | 与 `NewClient` 一致，可用 `httpx.CloseClient` 归还 idle 连接 |
| 失败判定 | 「连上了但 60s 不返回响应头」→ 报错；「传输中途被上游掐断」→ 报 net error（进入既有分类/重试） |

驱动侧契约：

```go
// drivers/115_Open（新增，等价上游）
const ossUploadAttempts = 3
func newOSSUploadHTTPClient(base *http.Client) *http.Client   // = httpx.NewStreamingClient(base, 60*time.Second)
func (d *Driver) ossUploadHTTPClient() *http.Client           // 未初始化时回退 d.client（防御）
func (d *Driver) ossUploadPartWithRetry(...) (string, error)   // seek → 试 → 可重试则退避重试
func isRetryableOSSUploadError(err error) bool
```

## 3. 数据流

```
upload.Manager worker
  └─ file.Service.UploadLocal (ctx: 任务取消/暂停可打断)
      └─ drivers/115_Open.Upload (ctx)
          ├─ API 调用（创建/查 token/complete）→ d.client（600s 总超时，不变）
          └─ 数据面 PUT
              ├─ ossSinglePartUpload（≤512MB 整文件）→ d.ossUploadHTTPClient()（无总超时）
              └─ ossMultipartUpload 每个分片（512MB）
                  └─ ossUploadPartWithRetry（3 次；退避 1s/2s；凭证错不重试）
                      └─ ossUploadPart → d.ossUploadHTTPClient()（无总超时）
      └─ drivers/189Cloud.Upload (ctx)
          ├─ init/getMultiUploadUrls/commit → d.client（30s，不变；含既有重试）
          └─ 分片 PUT → d.uploadClient = NewStreamingClient(d.client, 60s)（无总超时 + 连接复用）
              └─ 既有 putUploadPartOnce 重试语义不变
```

## 4. 关键决策与取舍

1. **为什么是「无总超时 + 响应头超时」而不是「把 600s 调大」**：总超时是「整请求」的上限，任何有限值都只是把失败阈值往后推；而数据面的正确约束是「有数据流动就允许继续」+「无响应即判死」。`ResponseHeaderTimeout` 恰好表达后者，且与上游实现一致（可减少未来移植冲突）。
2. **残留风险与缓解**（明确记录，不隐藏）：若上游**接收了连接但长时间不返回响应头且不报错**，则由 60s 响应头超时兜底；若上游**开始返回响应头后停止读 body/停止发送**，Go 的 `http.Transport` 无写超时，理论上可长时间阻塞。缓解：① 任务是可取消的（暂停/取消经 `ctx` 打断，`Drop` 关闭 idle 连接）；② `http.DefaultTransport` 克隆带 TCP keepalive（30s）可探测死连接；③ 该风险与上游一致，且仅发生在数据面（API 面仍有 600s/30s 兜底）。若后续需要更强的兜底（例如按「无字节进展」计时），另建任务处理。
3. **115 单请求上传为什么不加重试**：与上游一致（上游只给分片路径加）。单请求上传（≤512MB）失败后由任务层重试更安全（可避免重复回调/重复计费语义），且下游有 `uploadConfirmAttempts` 等既有机制。保持与上游同构，降低未来移植成本。
4. **115 凭证错误不纳入重试**：`isOSSCredentialError` 已有一条「刷新 token 后重试一次」的路径，若在 `ossUploadPartWithRetry` 内也重试，会形成「3×刷新」放大；故重试函数对凭证错误直接返回，交由调用方既有逻辑处理。
5. **重试与进度**：`uploadProgressReader` 按读到的字节上报进度；重试会重新 `Seek` 并重新读同一段数据，因此**必须在重试前把进度基准复位**（重试函数采用「先 Seek，再调用 `ossUploadPart`」，`ossUploadPart` 内部以 `base=uploadedOffset` 为基准构造 reader，天然不会重复累加；测试用假 transport 断言最终一次调用计数与 offset）。

## 5. 上游 `869974b` 适配差异表

| 上游改动 | 本方处置 | 理由 |
|---|---|---|
| `internal/httpx.NewStreamingClient` 新增 | **原样移植** | 共享设施，零分叉 |
| 115 `uploadClient` + `ossUploadHTTPClient()` + 数据面改用 | **原样移植** | 同上 |
| 115 `ossUploadPartWithRetry` / `isRetryableOSSUploadError` | **原样移植** | 同上 |
| 189 `uploadClient` → `NewStreamingClient(d.client, 60s)` | **原样移植** | 同上 |
| 189 `Config.InternalExperimental = true` | **不移植** | 这是上游给「实验驱动」做的 UI 标记；189 在本方是主力驱动，不应标实验 |
| 139/123/Baidu/Guangya/OneDrive/Quark 六驱动同类改动 | **不适用** | 本方精简分支仅保留 115/189/LocalFs |
| 115 API 客户端超时（上游 30s） | **保留本方 600s** | 本方 0.0.3 起的既有定制（`4d8e868`），本任务不改 |
| 115 分片大小（上游自适应 20MB 起） | **保留本方固定 512MB** | 用户 2026-08-26 定制；本任务不改（数据面无总超时后 512MB 分片才真正可行） |
| `internal/file/service.go` 日志字段 `err`→`error` | **不移植** | 与本源日志字段约定不同，且会改动我方 0.0.32 冷却日志行附近的代码，无收益 |

## 6. 兼容性与回滚

- **接口兼容**：`NewStreamingClient` 为新增导出函数，无调用方受影响；`Driver` 结构体新增私有字段，无接口变更；数据库/API/前端契约零变化。
- **行为变化**（预期）：115/189 上传在大文件或慢链路下不再被总超时掐断；115 分片对瞬时故障最多额外尝试 3 次（每次退避 1s/2s）。
- **回滚**：单次提交可 revert（`git revert`）；镜像回退 `ghcr.io/zhemed/litepan:v0.0.37`；本地容器用既有 `docker run` 命令改 tag 即可。无数据迁移、无 schema 变更。
