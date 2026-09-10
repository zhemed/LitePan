# fix-resume-freeze-batch-endpoint

## Goal

修复"点击继续上传卡死网页"（用户报告，0.0.25 发布时已实现）。补记任务档案。

## 根因（实测）

- 前端 resume 模式：`for (1620 个暂停任务) { await resumeUploadTask() }`——每次 HTTP + 响应式 patch（7814 条任务数组）+ 起调度器，串行风暴阻塞主线程
- 后端只有单任务 resume 端点（无批量），前端只能逐个发
- 叠加因素：任务列表 API 无分页/过滤，7814 条全量 5.4MB（页面加载/刷新重）

## 修复（0.0.25）

1. `Manager.BatchResume`：一次恢复整批（内部仍逐个走 Resume 的排队与并发闸门）
2. `POST /files/upload/tasks/batch-resume`（镜像 batch-pause）
3. 前端继续上传：远程任务单次 batchResume + 一次刷新；本地（浏览器内）任务保持逐个恢复
4. 测试：批量恢复去重/缺失登记/确定性短路；端点实测 3 任务恢复成功

## Acceptance Criteria

- [x] 端点实测通过（updated_task_ids 返回、任务转 success）
- [x] vet/全模块测试/类型检查/构建全绿
- [x] 0.0.25 三 tag + release + 部署（digest 44999230）
- [ ] 归档 + journal + push

## 遗留优化（另议）

任务列表 API 分页/过滤 + 状态汇总（7814 条 5.4MB 全量返回对页面加载与 SSE 快照都重）。
