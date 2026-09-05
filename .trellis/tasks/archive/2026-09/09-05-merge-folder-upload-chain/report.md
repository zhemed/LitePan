# 合入A路文件夹上传链并发版0.0.15 · 执行报告

执行时间：2026-09-05 · commit `d972292` · tag `v0.0.15` · GHCR `sha256:a85dcc0`（105MB）

## 结论

**已合入并发布 `0.0.15`，容器运行新镜像且健康，`migration 0022` 在线生效（`batch_id/batch_name` 列 + 索引已落库）。** 待用户 UI 验证文件夹上传。

## 依赖判定（执行前完成，记录）

1. `local_upload.go` hunk 依赖 {types batch字段、domain、0022、repo、persist}——全在 PASS 集，无 B 依赖 → **0022 与 repo 列入合入**（否则 Upsert 写 batch 列会 SQL 报错）。
2. `folderPlanner` 引用的 store 函数本地独缺 `addLocalUploadTasks` → 从 store 文件拆 hunk 移植。
3. `panelActions` hunk 引用本地不存在的 `activeRelayCount`（中继已删）→ 整 hunk 排除（本地行为等价）。
4. `taskStream` 为 B-road SSE 客户端 → 排除，保留本地轮询。
5. `useUploadBatchActions` 70行含 `handleToggleUploadTasks`（B）→ 排除；`useUploadTasks` 去掉该导出单行后合入。
6. `enqueueUploadFolderFiles` 定义在新 planner 内，经 `useUploadTaskActions` 透传，链路完整。

## 合入集（17 白名单 + 2 拆分）

- 后端：`local_upload.go/types.go/domain/repo/persist/0022/local_upload_test.go`
- 前端：面板拖拽（`webkitGetAsEntry` 递归限单个文件夹）、planner、dispatcher（含万级文件内存释放）、fileInput、confirmUpload 文案、FileBrowser 接线、types×2、formatters、tasks（去 toggle 行）、store 11 hunk
- 排除：B-road 全套（TaskPanel/SSE/manager-struct/BatchPause/worker/router行/delete扩展/api-upload批量接口/panelActions/stream/uploadTaskTree/测试脚本）

## 验证

- `vue-tsc` 0 错误；`go build` 0；`upload/api/store` 测试 ok（含新 `TestTrimUploadBatchRoot`）。
- 全量套件仅基线已知 `internal/file` 中文数字 2 用例失败（与本次无关）。
- `v0.0.15` 三标签已推，`github/main` 同步，容器 `36f46d52` 健康。

## 待用户 UI 验证

1. 从服务器上传面板拖入单个文件夹 → 确认弹窗显示文件夹名与文件数 → 上传后网盘目标目录结构一致。
2. 空文件夹拖入应提示"空文件夹暂不支持"；一次拖多个文件夹应提示"一次只能拖入一个"。
3. 大批量（>1000 文件）上传时任务列表可用、浏览器内存不暴涨。
4. 删除文件夹任务弹窗文案为"删除文件夹任务/同时删除网盘中的文件夹"。
