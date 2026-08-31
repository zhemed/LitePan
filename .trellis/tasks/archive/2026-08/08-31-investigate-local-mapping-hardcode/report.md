# 报告：本地映射（多选）是否为硬编码

**任务**：`08-31-investigate-local-mapping-hardcode`  
**截图**：`24500b 137x39 本地映射（多选）`  
**基线**：`main 7439a18 v0.0.5`  
**结论**：**不是硬编码，已动态化**（`fetchLocalMappings` 拉 `GET /api/admin/tools/local-upload/config`）

---

## 一、现状（0.0.5）

| 文件:行 | 代码 | 含义 |
|---|---|---|
| `web/src/components/admin/AutomationPanel.vue:663` | `const localMappingOptions = ref([])` | 空 `ref`，非常量数组 |
| `664-669` | `fetchLocalMappings = async () => { fetch('/api/admin/tools/local-upload/config')... mappings.map(m=>{value:m.name}) }` | 动态拉取 |
| `461` | `<AppSelect v-model="configAction.params.mappings" :options="localMappingOptions" multiple />` | 下拉多选绑定动态 `options` |
| `986` | `if (targetAction?.type==='local_upload') void fetchLocalMappings()` in `openConfig` | 打开配置时触发加载 |
| `internal/api` | `GET /api/admin/tools/local-upload/config` → `settings KeyLocalUploadMappings` `[{name,path}]` | 后端动态：工具箱→本地上传的映射配置（`我的文件/杂物间/pve_backup` 等用户自增，非写死） |

**验证**：`grep -n "localMappingOptions = \[" AutomationPanel.vue ==0`（无硬编码 `['我的文件']`），`grep -n "fetchLocalMappings" ==2`。

**后端来源**：`internal/settings/registry.go KeyLocalUploadMappings` 为 `local_upload_mappings` JSON，`internal/api/tools/local-upload` 的 `Config` 接口返回 `mappings`，前端直接映射 `m.name`。

---

## 二、历史演进（硬编码 → 动态）

| 补丁 | 时间 | 变更 | 硬编码？ |
|---|---|---|---|
| `283b875 feat: support multi-select` | 2026-08-26 11:35 | `service_run` 支持 `mappings[]` 循环；前端 `AutomationPanel` 改 checkbox 多选，但 `localMappingOptions = [{value:'我的文件'},...3]` **硬编码** 3 值 | **是**（3 值写死） |
| `2b44969 fix: make mapping multi-select dynamic` | 2026-08-26 11:40 | `localMappingOptions = ref([])` + `fetchLocalMappings()` 拉 API；checkbox → `AppSelect multiple` | **否**（动态）|
| `be39a6a adapt local_upload from LitePan-own` | 2026-08-30 | 移植到 `LitePan` 时即带 `ref([])+fetch` 动态版 | 否 |
| `0.0.5 7439a18` | 2026-08-31 | 保持动态，仅 `service_schedule` 修复 | 否 |

**来源**：`_extracted/.../patches/0003` 与 `0004` diff、`git show 2b44969`。

`0003` 硬编码片段：
```js
const localMappingOptions = [
  {value:'我的文件',label:'我的文件'},
  {value:'杂物间',label:'杂物间'},
  {value:'pve_backup',label:'pve_backup'}
]
```
`0004` 后：
```js
const localMappingOptions = ref([])
const fetchLocalMappings = async () => {
  const data = await fetch('/api/admin/tools/local-upload/config').then(r=>r.json())
  localMappingOptions.value = data.mappings.map(m=>({value:m.name,label:m.name}))
}
```

---

## 三、是否为硬编码判定

- **当前 0.0.5**：**否**。`localMappingOptions` 为 `ref([])`，运行时 `fetch`，与用户在 `工具箱 → 本地上传` 新增/删除的映射实时一致，新增第 4 个映射无需改代码。
- **若看到仍为 3 固定值**：可能是浏览器缓存 `internal/api/web` 未更新（需 `docker pull 0.0.5` + 硬刷新）或后端 `KeyLocalUploadMappings` 仅 3 条。

---

## 四、建议

- 无需再改；如需验证：`curl -s http://IP:5211/api/admin/tools/local-upload/config -b cookies | jq .mappings` 应返回当前映射数，与下拉一致。
- 若需展示“已选/可用”数，可在 `field-tip` 中加 `已加载 {{localMappingOptions.length}} 个`（非必须）。

---

## 附：取证

```bash
grep -n "localMappingOptions\|fetchLocalMappings" web/src/components/admin/AutomationPanel.vue
sed -n '663,670p' web/src/components/admin/AutomationPanel.vue
git show 283b875:web/src/components/admin/AutomationPanel.vue | grep -A5 "localMappingOptions"
curl -s http://127.0.0.1:5211/api/admin/tools/local-upload/config | head
```
