# implement.md — 09-12-fix-upload-timeout-streaming-client

执行顺序固定，每步末尾给验证命令与判据；判据不满足即停下报告（不绕过）。

## 0. 前置（已完成）
- `task.py create` → `prd.md` / `design.md` / `implement.md` → `set-scope complex` → `flow_gate.py pre-start` → `task.py start`。

## 1. `internal/httpx`：流式客户端
1. `client.go` 新增 `NewStreamingClient(base *http.Client, responseHeaderTimeout time.Duration) *http.Client`（实现按 `design.md` §2 契约；与上游 `869974b` 同构）。
2. 新增 `client_test.go`：
   - `Timeout == 0`；
   - 传入带 `Proxy`/`DisableCompression` 的 base → 克隆后属性保留（且 base 自身未被修改）；
   - `ResponseHeaderTimeout == 60s`；`<=0` 时不设置；
   - `base == nil` / 非 `*http.Transport` → 可正常返回且 `Timeout == 0`；
   - `CloseClient` 不 panic。

**判据**：`GOWORK=off go test ./internal/httpx/ -count=1` 通过。

## 2. `drivers/115_Open`：数据面专用客户端 + 分片重试
1. `driver.go`：`Driver` 增 `uploadClient *http.Client`；`Init` 中 `if d.uploadClient == nil { d.uploadClient = newOSSUploadHTTPClient(d.client) }`；`Drop` 中 `httpx.CloseClient(d.uploadClient)`。
2. `upload.go`：
   - 新增 `const ossUploadAttempts = 3`、`newOSSUploadHTTPClient`、`(d *Driver) ossUploadHTTPClient()`；
   - `ossSinglePartUpload`、`ossUploadPart` 的 `d.client.Do(req)` → `d.ossUploadHTTPClient().Do(req)`；
   - 新增 `isRetryableOSSUploadError`（表驱动覆盖：`net.Error`、unexpected eof、connection reset、broken pipe、HTTP 429/500/502/503/504；反例：400/403/404、凭证错误、上下文取消）；
   - 新增 `ossUploadPartWithRetry`（3 次；退避 `attempt * 1s`，`select` ctx 可取消；每次尝试前 `f.Seek(uploadedOffset, io.SeekStart)`；`ctx.Err() != nil`、凭证错误、不可重试错误直接返回）；
   - `ossMultipartUpload` 分片调用改用 `ossUploadPartWithRetry`，并删掉调用点原有的重复 `Seek`（重试函数内已做）。
3. 新增 `drivers/115_Open/upload_retry_test.go`：
   - `isRetryableOSSUploadError` 表测；
   - `ossUploadPartWithRetry` 用假 `http.RoundTripper`（首次返回可重试错误、二次成功）→ 断言返回 ETag 且请求次数=2、请求体字节数正确（`ContentLength`/读取长度）；
   - 连续可重试错误 → 恰 3 次尝试后返回最后错误；
   - 不可重试错误（如 HTTP 403）→ 仅 1 次尝试；
   - 测试不得访问真实网络（无域名解析、无外网）。

**判据**：`GOWORK=off go test ./drivers/115_Open/ -count=1` 通过；`git diff drivers/115_Open/driver.go` 中 `600 * time.Second` 未被改动。

## 3. `drivers/189Cloud`：上传客户端流式化
1. `driver.go`：`uploadClient` 构造改为 `httpx.NewStreamingClient(d.client, 60*time.Second)`；移除 `DisableCompression/DisableKeepAlives` 写法；确认 `time` 与 `httpx` import 仍被使用（`go vet` 兜底）。
2. 不改分片大小、`putUploadPartOnce`/`retryableUploadURLFailure`/`maxAttempts` 与账号节流。

**判据**：`GOWORK=off go test ./drivers/189Cloud/... -count=1` 通过；`grep -n "DisableKeepAlives" drivers/189Cloud/*.go` 无输出。

## 4. 记录更正
1. 在 `design.md` §5 与本任务 journal 写明：09-12 调查报告 P1 的「30s」表述有误，实为 600s（`4d8e868`/0.0.3 起），问题实质＝API/数据面共用客户端 + 缺分片重试。
2. 在 `.trellis/spec/backend/backend/driver-development.md` 增补一条约定：**数据面上传 PUT 使用 `httpx.NewStreamingClient`（不设总超时、`ResponseHeaderTimeout=60s`），API 调用使用 `httpx.NewClient`（设总超时）**，附本任务与上游依据。

**判据**：spec 段落存在且与代码一致（引用函数名/文件路径准确）。

## 5. 质量门（全绿才继续）
```bash
GOWORK=off go vet ./...
GOWORK=off go test ./... -count=1
GOWORK=off go build ./...
cd web && npm run type-check && npm run build && npm run check:memo
```
**评审门**：`git diff --name-only` 必须落在 PRD/design 声明的文件集内（`internal/httpx/client*.go`、`drivers/115_Open/*`、`drivers/189Cloud/driver.go`、`README.md`、`docker-compose.yml`、`.trellis/**`）。若出现 `web/src` 源码改动或其它文件改动 → 停下报告。

## 6. 发版 0.0.38
```bash
# 6.1 版本号
# README.md 两处 + docker-compose.yml 一处 → v0.0.38
# 6.2 镜像
docker build -t ghcr.io/zhemed/litepan:0.0.38 -t ghcr.io/zhemed/litepan:v0.0.38 -t ghcr.io/zhemed/litepan:latest .
docker push ghcr.io/zhemed/litepan:0.0.38 && docker push ghcr.io/zhemed/litepan:v0.0.38 && docker push ghcr.io/zhemed/litepan:latest
# 6.3 提交与 tag
git add -A && git commit -m "fix(driver): stream upload bodies and retry 115 parts, bump to 0.0.38"
git push github main
git tag v0.0.38 && git push github v0.0.38
gh release create v0.0.38 --repo zhemed/LitePan --title "v0.0.38" --notes "<变更摘要>"
# 6.4 本地部署（沿用既有命令）
docker rm -f litepan && docker run -d --name litepan --restart unless-stopped -p 5211:5211 \
  -e TZ=Asia/Shanghai -e LITEPAN_LOG_LEVEL=info \
  -v /root/LitePan/data:/app/data -v /root/LitePan/mounts:/app/mounts:shared \
  --device /dev/fuse --pid host --privileged ghcr.io/zhemed/litepan:latest
```
**判据**：三 tag 同 digest；`/api/health` OK；form 表单登录成功；`/api/files/upload/tasks/summary` 返回正常。

## 7. 收尾
1. `skill trellis-check`（清单核对） → `flow_gate.py mark-check`（附检查摘要）。
2. 勾选 `prd.md` 验收项 → `flow_gate.py pre-archive` → `task.py archive --skip-branch-validation`。
3. `add_session.py`（journal，含记录更正） → `git push github main`。

## 回滚点
- 步骤 1–3 后：`git checkout -- <files>`；
- 步骤 6.2 后：镜像保留 `v0.0.37` 可回退；
- 步骤 6.4 后：用 `v0.0.37` 重建容器即可（无数据迁移）。
