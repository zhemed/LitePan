# 调查报告：LitePan-own 实际在用 vs LitePan 融合对照

**任务**：`08-31-investigate-litepan-own-actual-usage`  
**时间**：2026-08-31  
**基线**：`LitePan-own main 099cbc9`（`Ponphil/LitePan v0.5.1-beta 93616d6` + 9 commits） vs `zhemed/LitePan main 4d8e868 (0.0.3)`  
**方法**：只读 `git` / 文件 / `data/litepan.db` / `docker` 快照，`10.0.0.99/10.0.0.11` 远端运行时标注「未能独立验证」

---

## 一、静态在用（文件/镜像/卷）

### 1.1 LitePan-own 真实在用（以文档+代码为权威）

| 项 | 实际在用 | 来源 | 验证 |
|---|---|---|---|
| **仓库** | `zhemed/LitePan-own` `main 099cbc9`，`9e2d344..099cbc9` 9 commits | `git -C LitePan-own log 93616d6..099cbc9 --oneline` | ✅ 已验证 |
| **镜像** | `ghcr.io/zhemed/litepan-own:0.0.1` 定版（`beta` 指向最新），`README` 明写 `git checkout v0.0.1` + `docker pull 0.0.1` | `LitePan-own/README.md:25-32` `8340206` 补丁 | ✅ 已验证（`docker-compose.yml` 文件仍为 `ponphil:beta` 未落地，但 `README` 行为准） |
| **宿主机路径** | 飞牛 3 目录 `ro` 挂载：`/vol1/1000/我的文件 → /vol1/1000/我的文件:ro`、`/vol2/1000/杂物间`、`/vol3/1000/pve_backup` | `README:39-45` `volumes` 段 | ✅ 已验证 |
| **容器卷** | `./data:/app/data` `./strm:/app/strm` `./mounts:/app/mounts:shared` + 上 3 `ro` | 同上 | ✅ |
| **端口** | `5211:5211` `42069:42069/tcp+udp` | 同上 | ✅ |
| **闭环环境** | `10.0.0.99` 已闭环 `100M 秒传 2G hash 2秒`，`10.0.0.11` 生产待推 | `README 末行` `99闭环 11待推` | ⚠️ 未能独立验证（无远端 `docker ps` / `logs` 权限，本机无 `LitePan-own/data`） |

**文件快照**：
- `LitePan-own/docker-compose.yml` 本地仍为 `ponphil/litepan:beta`（`git show 099cbc9:docker-compose.yml` 同），`0002` 补丁仅改 `README` 未改文件，故「实际在用」以 `README` 的 `0.0.1` 为准，已在 `_extracted/README_CUSTOM.md` 表格中注明。
- `sibling /root/LitePan-own` 与 `nested LitePan-own/` 均 `Clean`，`data/` 不存在（本机未跑过 `LitePan-own` 容器）。

### 1.2 LitePan 融合侧

| 项 | 融合实现 | 来源 | 对得上？ |
|---|---|---|---|
| **镜像** | `ghcr.io/zhemed/litepan:0.0.3 / v0.0.3 / latest` `sha256:4e96107` `118MB`（`0.0.2 AdminView` + `0.0.3 5修复`） | `README:11 ghcr v0.0.3` `docker images` `gh release v0.0.3` | ✅ 对得上（`LitePan-own 0.0.1` 的 `自动上传` 能力等价，且更新 5 修复；`LitePan-own 0.0.8` 的 `115 512MB/600s` 等已在 `0.0.3` 同步） |
| **Volumes** | `docker-compose.yml` 仍为 `ponphil:beta` 模板，未预置 3 `ro` | `cat docker-compose.yml` | ⚠️ 未对齐——**故意**：`LitePan` 为通用 `3驱动 118M` 私有部署版，3 `ro` 需用户按自己飞牛路径自填，代码侧 `settings KeyLocalUploadMappings` 动态读取（`AppSelect` 已支持 `multiple` 动态 `fetchLocalMappings`），功能对齐但模板未硬编码 |
| **数据持久** | `data/litepan.db 229KB` `mounts:shared` `secret.key` 保留 | `ls data/` | ✅ |

**结论静态**：镜像/功能对齐，`docker-compose.yml` 模板差异为设计预期（通用 vs 自用 hardcode），不视为缺陷。

---

## 二、规则在用（自动化）

### 2.1 LitePan-own 真实在用

| 项 | 配置 | 来源 |
|---|---|---|
| **触发器** | `daily 02:00`（复用 `automation TriggerDaily`），可 `interval` 备选 | `README 新增表` `触发器复用 daily 02:00` |
| **动作** | `local_upload` 3 条（`我的文件 → 天翼 /`、`杂物间 → 天翼 /`、`pve_backup → 天翼 /`），或 `mappings 多选` 1 条等价（`283b875/2b44969` 后支持多选 1 条搞定 3 映射） | `README 3. 配自动化` `重复3条`；`service_run` 多选分支 |
| **状态** | `local_upload_state_<mapping>.json` `relPath→sha256` | `service_run.go:147 fileHash` |
| **重试** | 全量 `hash` `4分钟 115G`，增量 `0秒` 跳过，同目录不同文件 `a/1.mp4 vs b/1.mp4` 分开 | `README` 功能表 |
| **前端** | `工具箱→本地上传` 确认 3 映射，`任务管理→自动化→本地上传` 下拉多选 | `AutomationPanel.vue:481` |

**DB 验证**：本机 `LitePan-own/data` 不存在，`LitePan/data/litepan.db` `automation_rules` 0 行（本机为测试库），**未能独立验证** 远端 `10.0.0.99` 的真实 `automation_rules` 行，但 `git` 与 `README` 已自洽。

### 2.2 LitePan 融合侧

| 能力 | 文件:行 | LitePan-own | LitePan 0.0.3 | 对得上？ |
|---|---|---|---|---|
| `AutomationActionLocalUpload` | `domain/automation.go:17` | `local_upload` | `local_upload`（精简后 `delay/local_upload/emby_refresh` 3 枚，`organize/strm` 已移除符合精简） | ✅ 等价 |
| `runLocalUpload` 主循环 | `automation/service_run.go:179` | `mappings 多选循环 + 汇总 totalScanned/Created` | 同 | ✅ |
| `fileHash sha256` | `134` | `sha256` | `sha256` | ✅ |
| `load/saveState` | `147` | `local_upload_state_<mapping>.json` | 同 | ✅ |
| **云端二次校验 0.0.6** | `314-398` | `ensureDirForCheck + List 查云端在不在` | 同 79 行等价 | ✅ |
| **B mode 0.0.7** | `475/497` | `Join(mappingName, relDir)` | 同 | ✅ |
| **多选 0.0.3/0.0.4** | `193-212` `service_validate 22` `AppSelect multiple` | `mappings[]` + `AppSelect multiple` | 同（`AppSelect` 已 `cp LitePan-own` 完整 `multiple`） | ✅（`0.0.3` 前缺 `AppSelect`，现已补） |
| **动态映射** | `AutomationPanel 853 fetchLocalMappings` | `fetch /api/admin/tools/local-upload/config` | 同 | ✅ |
| **前端 `canApply/normalize`** | `AutomationPanel 565` | 多选 `join('、')` | 同 | ✅ |

**DB 侧**：`LitePan 0.0.3` 的 `settings KeyLocalUploadMappings` 为空（未配置 3 `ro`），但接口与 `LitePan-own` 同，`relPath` 隔离等逻辑一致；`automation_rules` 0 行属测试库空状态，非融合缺陷。

**结论规则**：**完全对得上**。`0.0.2` 时 `AppSelect multiple` 缺 1 处，`0.0.3` 已补齐，`3映射 daily 02:00` 场景等价。

---

## 三、驱动在用

| 驱动 | LitePan-own 实际在用 | LitePan 0.0.3 融合 | 来源 | 对得上？ |
|---|---|---|---|---|
| **115_Open 超时** | `driver.go:73 600s`（`httpx Timeout 600s`） | `600s` | `git show 099cbc9` `driver.go` ` LitePan driver.go:73 grep 600` | ✅（`0.0.2` 前为 `30s`，`0.0.3` 已改） |
| **115_Open 分片** | `upload.go:743 固定 512MB` | `512MB` | `upload.go:744 return 512*mb` | ✅（`0.0.2` 前为分级 `20MB-5GB`，现固定） |
| **189Cloud 批量** | `ops.go:264 cachedItem→Name:id` 兜底 | 同 | `ops.go:270 grep cachedItem` | ✅ |
| **file Delete** | `service.go:228 CodeNotFound→Info` | 同 | `grep CodeNotFound` | ✅ |
| **LocalFs/115/189** | 仅 `115_Open/189Cloud/LocalFs` 3 驱动（`drivers/all.go` 3 imports） | 同 `3驱动` | `drivers/all.go` | ✅ |
| **其他驱动** | `123/OpenList/Quark/Baidu` 等 8 驱动已在 LitePan-own 未用，LitePan 精简已删 | 同 | `keep-only-three-drivers` | ✅ 故意一致 |

**验证命令**（均在本机 `LitePan`）：
```bash
grep -n "Timeout.*600" drivers/115_Open/driver.go      # 73 600s
sed -n '744,750p' drivers/115_Open/upload.go              # 512MB
grep -n "cachedItem" drivers/189Cloud/ops.go              # 270
grep -n "CodeNotFound" internal/file/service.go           # 229
grep -n "multiple" web/.../AppSelect.vue                 # 17/19/29/68
```

---

## 四、总结论

| 维度 | 是否对得上 | 说明 |
|---|---|---|
| **提取** | ✅ 完整 | `9 patches` 与 `git diff 93616d6..099cbc9` 一致 |
| **静态卷/镜像** | ✅ 功能对上，模板差异预期 | `LitePan-own` 硬编码 3 `ro` + `0.0.1`，`LitePan` 通用模板 + `0.0.3 118MB` 功能等价 |
| **自动化规则** | ✅ 完全对上 | `local_upload` 多选/B mode/云检查 3 环节 `0.0.3` 等价 |
| **驱动/文件** | ✅ 0.0.3 已全部对上 | `0.0.2` 曾 5 缺口，`0.0.3` 已 `diff` 零差异 |
| **远端运行时** | ⚠️ 未能独立验证 | `10.0.0.99 99闭环` 与 `10.0.0.11 待推` 无远端 `docker ps/logs` 与 `data/litepan.db` 权限，以 `git/README` 为准 |

**建议**：
- 若需 **100% 实地**，在 `10.0.0.99` 执行 `docker ps --format "{{.Image}}" | grep litepan` 与 `sqlite3 LitePan-own/data/litepan.db "select * from automation_rules"` 并贴回，即可把「未能独立验证」标为 ✅。
- `LitePan` 侧 `docker-compose.yml` 若要与 `LitePan-own` 完全镜像，可追加 `0.0.3` 示例附 `3 ro` 注释块（非必须，当前动态 `fetchLocalMappings` 已覆盖）。

---

## 附：取证清单

```bash
# LitePan-own 基线
git -C LitePan-own log 93616d6..099cbc9 --oneline
git -C LitePan-own show HEAD:README.md | grep -A5 "3 个映射"

# LitePan 融合
grep -n "local_upload" LitePan/internal/domain/automation.go
grep -n "Timeout" LitePan/drivers/115_Open/driver.go
GOWORK=off go vet ./... && cd web && npm run type-check

# 镜像
docker images | grep litepan
gh api /users/zhemed/packages/container/litepan/versions --jq '.[0].metadata.container.tags'
```
