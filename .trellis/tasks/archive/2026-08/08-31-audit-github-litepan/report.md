# 巡检报告：GitHub `zhemed/LitePan` 远端

**任务**：`08-31-audit-github-litepan`  
**时间**：2026-08-31 17:27 UTC  
**对比基线**：本地 `main 66337e3 v0.0.3 sha256:4e96107 118MB`  
**远端**：`https://github.com/zhemed/LitePan` `main 66337e3` `PUBLIC`  
**方法**：`gh api / curl raw / git ls-remote / docker pull` 只读

---

## 一、远端元数据

| 项 | 远端 | 来源 | 结论 |
|---|---|---|---|
| **仓库** | `zhemed/LitePan` `PUBLIC` `main` `fork:false archived:false private:false` | `gh repo view --json visibility,defaultBranchRef` `{"visibility":"PUBLIC","defaultBranchRef":{"name":"main"}}` | ✅ |
| **描述** | `LitePan - 精简版云盘聚合（仅保留115/天翼云盘/本机存储，已移除STRM/共享/缓存整理/增强工具/跨盘秒传等）` | 同上 `description` | ✅ 与 `0.0.3` 精简一致 |
| **大小/语言** | `size 73833 (72MB git)` `language Go` `stars 0 forks 0 issues 0` | `gh api repos/zhemed/LitePan --jq {size,language}` | ✅ |
| **更新** | `updatedAt 2026-08-30T17:25:51Z` `pushed_at 2026-08-30T17:25:34Z` | 同上 | ✅ 与 `0.0.3` push 时间 `17:06` 后 `organize 17:25` 一致 |
| **分支** | `git ls-remote` `HEAD 66337e3` `refs/heads/main 66337e3` | `git ls-remote https://...` | ✅ 与本地 `66337e3` 一致 |
| **落后检查** | `git fetch --dry-run` 无输出；`git log github/main..HEAD` `0`，`HEAD..github/main` `0` | `git log` | ✅ 完全同步 |

---

## 二、文件（`contents` + `raw`）

| 文件 | 远端 `raw` 状态 | `sha256` 本地 vs 远端 | 结论 |
|---|---|---|---|
| `README.md` `1767B` | `HTTP 200` `cache-control max-age=300` | `00edf3aa...` 本地 `00edf3aa...` 远端 `00edf3aa...` **一致** | ✅ `ghcr v0.0.3` `115/天翼/本机` 3 驱动表正确 |
| `docker-compose.yml` `624B` | `200` | `c5e802ff...` 一致 | ✅ `image: ghcr.io/zhemed/litepan:v0.0.3` 已于 `61c07c6` 同步 |
| `docker-compose.fnos.yml` `381B` | `200`（via `contents`）|  同步 `v0.0.3` | ✅ |
| `install-docker.sh` `5148B 105行` | `200` `etag 70668...` `105` 行 | `f3ac4394...` 一致 | ✅ `29.7.2 + v5.4.0 全平台` 可 `curl \| bash` |
| `go.mod` `4506B` `module litepan go 1.26.6` | `contents` 存在 | - | ✅ |
| 顶层 `contents` | `.agents .dsh .trellis ACK/Dockerfile/docs/drivers/internal/pkg/web ...` 25 项 | `gh api contents` 列出 | ✅ 无缺失，`internal/api/web` 产物在 `git` 中 `5.7M 111文件` |

**取证**：`curl -fsSL https://raw.githubusercontent.com/zhemed/LitePan/main/README.md | sha256sum` 与本地 `sha256sum README` 一致；`install-docker.sh` 同；`docker-compose.yml` 同。

---

## 三、发布（Release / Tag）

| 项 | 远端 | 来源 | 结论 |
|---|---|---|---|
| `gh release list --limit 10` | `v0.0.3 Latest 2026-08-30T17:06:52Z` `v0.0.2 16:17:33Z` `v0.0.1 10:38:38Z` | `gh release list` | ✅ 3 个 `0.0.x` 稳定 release 连贯 |
| `v0.0.3 body` | `fix: 补齐 LitePan-own 5处修复（10.0.0.99闭环） -115 600s/512MB -189Cloud容错 -file NOT_FOUND -AppSelect multiple -镜像 sha256:4e96107` | `gh api releases/tags/v0.0.3` | ✅ 与 `0.0.3` 实际修复一致 |
| `tags` | `v0.0.3 4d8e868` `v0.0.2 6ef108b` `v0.0.1 452b0c9` + `v0.5.2/0.5.1/0.5.0/0.3.0/0.2.0-beta` 5 个旧 beta | `git ls-remote --tags` `gh api tags` | ✅ `v0.0.x` 3 个 + `beta` 5 个共 8，`HEAD` 同 `66337e3` |
| `draft/prerelease` | `v0.0.3 draft:false prerelease:false` | 同上 | ✅ 正式 release |

**一致性**：`git ls-remote refs/tags/v0.0.3` `4d8e868` 与本地 `git tag v0.0.3` 同。

---

## 四、镜像（GHCR）

| 项 | 远端 | 来源 | 结论 |
|---|---|---|---|
| **visibility** | `public` | `gh api users/zhemed/packages/container/litepan --jq .visibility` | ✅ 可匿名 `docker pull` |
| **versions** | `v0.0.3 0.0.3 latest 2026-08-30T17:06:20Z id 1188434590` `v0.0.2 0.0.2 2026-08-30T16:16:24Z` `v0.0.1 0.0.1 2026-08-30T10:28:08Z` `v0.5.2-Beta` | `gh api .../versions --jq tags` | ✅ 9 tags（`3个版本 x 2 形式 + latest + Beta`） |
| **digest** | `sha256:4e96107799d861e829325d3c31c535a73cad88f4deae769a0a501570fdd3f8e6` 三 tag 同 | `docker pull v0.0.3 Digest` `docker images` | ✅ 与本地 `ghcr.io/zhemed/litepan:0.0.3` `sha256:38a63... imageID` 同源（`docker pull Status: up to date`） |
| **pull** | `docker pull ghcr.io/zhemed/litepan:v0.0.3` `up to date` | 本机实测 | ✅ `118MB` `public` 无 `unauthorized` |
| **本地镜像** | `ghcr.io/zhemed/litepan:0.0.3/v0.0.3/latest` `118MB` 同 `digest` | `docker images | grep litepan` | ✅ |

---

## 五、一致性（本地 vs 远端）

| 检查 | 结果 | 结论 |
|---|---|---|
| `git rev-parse HEAD vs github/main` | `66337e3` vs `66337e3` 同 | ✅ 0 落后 |
| `git log github/main..HEAD` | `0` | ✅ |
| `sha256 README/docker-compose/install-docker` 3 文件 | 本地 `00edf3/c5e802/f3ac43` vs 远端 `raw` 同值 | ✅ 100% 一致 |
| `docker-compose.yml image` | `ghcr v0.0.3` 与 `README v0.0.3` 一致，远端 `raw` 同 | ✅（本地 `61c07c6` 已同步） |
| `Actions` | `gh api actions/runs` `[]` 无 workflow | ✅（本仓库无 CI，非缺陷） |
| `LICENSE` | `gh api license NOASSERTION` `LICENSE 4716B PolyForm NC` 本地同 | ✅ |

---

## 结论

**远端与本地 `v0.0.3` 100% 一致**：`main` 同 `66337e3`，`README/docker-compose/install-docker` `sha256` 同，`Release v0.0.3/v0.0.2/v0.0.1` 连贯，`GHCR 0.0.3/v0.0.3/latest` `public` `118MB` 同 `digest`，`visibility PUBLIC` 可匿名拉取，无落后、无漂移。

**建议**：保持 `0.0.3` 为稳定基线，后续 `0.0.4` 递增；若启用 `GitHub Actions` 可补 `build/publish` 自动化（非必须）。

---

## 附：取证命令

```bash
gh repo view zhemed/LitePan --json visibility,defaultBranchRef,updatedAt
git ls-remote https://github.com/zhemed/LitePan.git | grep HEAD
gh release list --repo zhemed/LitePan --limit 5
gh api repos/zhemed/LitePan/releases/tags/v0.0.3 --jq .published_at
gh api users/zhemed/packages/container/litepan/versions --jq '.[].metadata.container.tags'
curl -fsSL https://raw.githubusercontent.com/zhemed/LitePan/main/README.md | sha256sum
docker pull ghcr.io/zhemed/litepan:v0.0.3
```
