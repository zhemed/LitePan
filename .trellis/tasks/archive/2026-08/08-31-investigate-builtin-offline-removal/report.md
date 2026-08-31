# 报告：磁力与临时下载是否可彻底移除

**任务**：`08-31-investigate-builtin-offline-removal`  
**截图**：`c8584f 884x237` `磁力下载端口 42069 / 临时下载目录 data/builtin_offline`  
**基线**：`main 7439a18 v0.0.5` → `0.0.6 已移除 offline_download`  
**结论**：**可彻底移除**（与 `local_upload` 零耦合），**3 键 + 1 面板 + 1 依赖 + 42069 端口** 需清理

---

## 一、清单（`builtin_offline` 仅 9 处 go + 前端 1 面板）

| 层 | 文件 | 行 | 内容 |
|---|---|---|---|
| **settings** | `internal/settings/registry.go:17-19` | 3 键 | `KeyBuiltinOfflineTempDir` `KeyBuiltinOfflineMaxSpeedMB` `KeyBuiltinOfflineBTPort` |
|  | `registry.go:101,103` | 2 | `stringSpec TempDir "data/builtin_offline"` `intSpec BTPort "42069"` |
| **go.mod** | `go.mod:13` | 1 | `github.com/anacrolix/torrent v1.61.0`（`grep anacrolix` 0 引用，**已无代码使用**） |
| **docker** | `docker-compose.yml:8-10` `Dockerfile` `42069` | 3 | `ports 42069:42069/tcp+udp` `devices /dev/fuse`旁，注释“磁力/BT” |
| **前端 面板** | `web/src/components/upload/UploadTaskSettingsPanel.vue:11` | 1 面板 | `OFFLINE_KEYS tempDir/maxSpeed/btPort` + `offlineForm/offlineItems` + `commitBTPort/selectTempDir` + 模板 `磁力下载端口` `临时下载目录` 2 输入 |
| **前端 入口** | `web/src/components/admin/SystemSettings.vue:54-56` | 3 键名数组 | `["builtin_offline_temp_dir", ...]` 用于 `otherSettings` 过滤 |
| **后端 零** | `grep builtin_offline --include=*.go \| grep -v registry` | 0 | **无 go 业务代码使用**（`upload/manager.go` 的 `builtin_offline` 仅测试 `t.TempDir()/builtin_offline` 路径拼接，非配置读取） |

**总数**：`9 go`（3 键 + 2 spec + 1 go.mod + 3 docker） + `1 面板`（`UploadTaskSettingsPanel` 300 行中约 80 行属此）。

**与 `offline_download` 区别**：`offline_download` 为 **云盘离线（URL→云）**，已在 `0.0.6` 删 `106+40` 处；`builtin_offline` 为 **容器内 `anacrolix/torrent` 磁力/BT 下载到 `data/builtin_offline` 再 `offline_handoff` 上传**，两者独立但均非 `local_upload` 所需（`local_upload` 走 `upload.SourceTypeServerLocal` 本地文件直传）。

---

## 二、依赖图

```
SystemSettings → 其他设置 → 读取 settings builtin_offline_* → registry 默认 data/builtin_offline / 42069
UploadTaskSettingsPanel → commitBTPort/selectTempDir → 同上 3 键

Docker 42069/tcp+udp → 为 torrent DHT/BT 端口（注释“用于磁力/BT”）
go.mod anacrolix/torrent → 已无 import（grep 0），仅残留依赖

local_upload
  → upload.Manager SourceTypeServerLocal → 本地文件 sha256 → 无 builtin_offline
  → settings KeyLocalUploadMappings → 无 builtin_offline
```

**零耦合**：`local_upload` 不读取 `builtin_offline_temp_dir/bt_port`，不 `import torrent`。

---

## 三、能否彻底移除

| 判断 | 结论 | 依据 |
|---|---|---|
| **local_upload 是否依赖** | **否** | `grep local_upload` 与 `builtin_offline` 无交集 |
| **DB/配置** | **可删** | 3 键仅 `otherSettings` 展示，无触发器/任务依赖；删后 `settings.String()` 返回 `""` 需前端容错 |
| **torrent 库** | **可删** | `grep anacrolix` 0 引用，`go mod tidy` 后自动移除 |
| **42069 端口** | **可删** | 仅为 torrent DHT，若不用磁力则无需映射；`network_mode: host` 下 `ports` 无效，但 `fnos` 版仍有 |
| **前端面板** | **可删** | `UploadTaskSettingsPanel` 整面板为 “离线任务设置”，删后 `SystemSettings` 的 `otherSettings` 列表需同步 |

**安全条件**：确认 **不再使用磁力/BT 内置下载**（`magnet:?xt=...` → `data/builtin_offline` → 上传）。若仍需 `磁力→云盘` 则保留。

---

## 四、彻底移除清单（若执行）

**后端**：
- `internal/settings/registry.go:17-19` 3 键常量 + `101,103` 2 spec
- `go.mod:13` `anacrolix/torrent v1.61.0` + `go.sum` 对应 + `go mod tidy`
- `docker-compose.yml:8-10` `docker-compose.fnos.yml:8-10` `42069` 端口段（可选，`network_mode: host` 下可留）

**前端**：
- `web/src/components/upload/UploadTaskSettingsPanel.vue` 整文件（或仅删 `OFFLINE_KEYS/offlineForm/commitBTPort/selectTempDir` 模板 2 输入）
- `web/src/components/admin/SystemSettings.vue:54-56` 数组中 3 键名

**Docker**：
- `Dockerfile` 无需改（`42069` 仅 compose）

**DB**：`configs` 中 `builtin_offline_*` 3 行（删后首次启动 `registry` 默认不再写入，可 `DELETE FROM configs WHERE key LIKE 'builtin_offline%'`）

---

## 五、建议

| 方案 | 操作 | 适用 |
|---|---|---|
| **A 彻底移除** | 按 §四 删 3 键 + 面板 + go.mod + 42069，`go mod tidy` `type-check` 后 `0.0.7` | **推荐**：若仅用 `local_upload`，`105MB→~104MB`，`go.sum` -500 行 |
| **B 保留** | 不删 | 仍需 `磁力→builtin_offline→上传` |
| **C 部分** | 仅删前端面板保留后端键 | 不推荐，配置孤悬 |

**风险**：A 方案下 `torrent` 依赖移除后 `go vet` 0，`vite build` 无 `UploadTaskSettingsPanel` 引用则 `SystemSettings` 需同步过滤，否则空面板。

---

## 附：取证

```bash
grep -R -n "builtin_offline" --include="*.go" --include="*.vue" --exclude-dir=_extracted
grep -R "anacrolix" go.mod
grep -n "42069" docker-compose.yml
grep -n "UploadTaskSettingsPanel" web/src/components/admin/SystemSettings.vue
```
