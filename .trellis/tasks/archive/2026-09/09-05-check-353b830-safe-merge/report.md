# 353b830 安全合入评估报告

评估时间：2026-09-05（UTC+8）
上游提交：`353b8308`（09-03《修复从服务器上传目录错位问题》）
本地文件：`web/src/components/file/CloudLocalUploadPanel.vue:76`

## 结论

**可以安全合入，建议合入。** 风险等级：**低**。

- `git apply --check`：**PASS**，可干净应用，无冲突。
- 改动范围：仅 1 个前端文件，8 增 1 删，无后端、无 API、无 DB migration。
- 前置依赖：`mappingIndex`、`loadBrowse` 在本地均存在且语义一致，无需前置补丁。
- 与本地精简方向无冲突（该面板是本地唯一上传入口，未被精简）。

## 补丁内容（全文）

```diff
-      if (mappings.value.length > 0) await loadBrowse(0);
+      if (mappings.value.length > 0) {
+        // 保留上次选中的映射，但需同步浏览其目录并防止映射列表变化导致越界，
+        // 否则会出现下拉显示目录 A、内容却是目录 B 的错位。
+        if (mappingIndex.value >= mappings.value.length) {
+          mappingIndex.value = 0;
+        }
+        await loadBrowse(mappingIndex.value);
+      }
```

修复的 bug：面板每次打开都按索引 0 加载内容，但下拉框 `mappingIndex` 保留的是上次选择（如 2），导致“下拉显示目录 A、内容却是目录 B”的错位；且当映射列表变短时旧索引会越界。

## 逐项验证

1. **上下文匹配**：分叉点 `4c160d9`、补丁前像 `353b8308^`、本地三处 watch/getConfig hunk **逐字相同**（第 74–76 行）。
2. **中间提交无干扰**：`de83b46`（上传批次化）虽改过同一文件，但只动 `props.onEnqueueFolderFiles` 与 `onDrop` 拖拽区，未碰 watch 块（grep 无命中）。
3. **本地未改过该文件**：`git log 4c160d9..HEAD -- <file>` 为空，本地即分叉点版本，无本地冲突。
4. **语义一致**：
   - `mappingIndex = ref(0)`（本地 :37）存在；
   - `loadBrowse(idx)`（本地 :83）首行即 `if (!mappings.value[idx]) return;` 越界保护，补丁的 clamp 是双保险；
   - 面板内其余调用（`selectMapping`、`onMappingChange`、`openEntry`、`jumpCrumb`）全部使用 `mappingIndex.value`，补丁行为与之统一。
5. **行为变化**：仅影响面板打开瞬间。旧行为：永远显示索引 0 内容（下拉可能仍指旧值）；新行为：下拉指哪就加载哪，越界时回落 0（等价旧行为）。无其他调用方依赖“打开即重置为 0”的语义。
6. **回归面**：纯前端展示逻辑；最坏情况（极端负索引）`loadBrowse` 守卫直接返回空列表，与旧代码的空映射分支表现一致。

## 合入建议

- 方式：`git cherry-pick -n 353b8308` 或手动套用 8 行（等价，`apply --check` 已过）。
- 合入后验证：`vue-tsc` + 打开从服务器上传面板切换两个映射后关闭重开，确认下拉与内容一致。
- 发版：按版本基线递增一个小版本即可（是否发版由用户定，本任务不执行合入）。
- 另两个候选（`c7a424c` 认证、`de83b46` 批次化）改动大，不在此任务范围内，需单独建任务评估。

## 只读确认

全程只执行 `git show/diff/apply --check/grep`，未修改业务代码，未合入，未切换分支。工作区除本任务目录外干净。
