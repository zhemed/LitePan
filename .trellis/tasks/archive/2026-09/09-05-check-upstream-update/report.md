# 上游更新检查报告

检查时间：2026-09-05（UTC+8，约 19:20）
本地：`main @ b77c1f7`（2026-09-01，v0.0.12）
上游：`origin/main @ 374affd`（2026-09-05 18:11，v0.5.4-beta）
分叉点：`4c160d9`（2026-08-29 部分显示优化）

## 结论

作者**有更新**：分叉点之后上游新增 **20 个提交**（08-30〜09-05），并发布新标签 **v0.5.4-beta**。
建议：**不需要整体同步，仅 3 个提交值得人工评估是否 cherry-pick**，其余与本地精简方向冲突或无关。

## 数量关系

- `git rev-list --left-right --count HEAD...origin/main` = `178 20`
  - 本地领先 178（精简 STRM/离线/跨盘/WebDAV/通知等 + v0.0.1〜v0.0.12 发版）
  - 上游领先 20（见下表）
- 直接 merge **不可行**：会把已删除的 STRM/跨盘/离线/Emby/公告等模块全部带回来，与既定精简方向直接冲突。

## 上游 20 个提交分类

| 日期 | 提交 | 内容 | 与本地关系 |
|---|---|---|---|
| 09-05 | 374affd 秒传星图手机端优化 | 秒传/星图前端 | 无关（秒传星图已删） |
| 09-05 | ce0992b 优化后台响应 | dashboard 慢日志 | 可看但非必需 |
| 09-04 | c7a424c 修复一系列认证问题 | 115/189 等驱动 auth + 189 新增 auth_response | **相关**：含 115_Open、189Cloud 认证修复，本地保留了这两个驱动 |
| 09-04 | 9aa08fe 跨盘传输功能扩展 | cross_transfer | 无关（已删） |
| 09-04 | 906b7cb STRM清理元数据 | strm | 无关（已删） |
| 09-03 | 353b830 修复从服务器上传目录错位 | CloudLocalUploadPanel 映射越界错位 | **相关**：该文件本地保留且为唯一上传入口 |
| 09-03 | 87f4b2d 联动刷新目录改为清理账号缓存 | 联动/emby 相关 | 无关 |
| 09-03 | 6b3b379 整理分类支持未命中放指定文件夹 | classifyorganize | 无关（缓存整理已删） |
| 09-03 | 8e332f3 认证刷新优化 | oauth | 中性，可看 |
| 09-03 | f90d890 修复AI初始化不渲染 | aiorganize | 无关 |
| 09-03 | d08cd04 支持emby补全媒体信息 | embyproxy | 无关 |
| 09-02 | b2ecd58 fix115strm | strm+115 transport | 无关（STRM 已删；115 transport 那 2 行可单独看） |
| 09-02 | a9458b2 部分显示优化 | Guangya/announcement | 无关 |
| 09-02 | 2eb66b4 Releases v0.5.3 | 版本号 | 无关（版本线已分叉：作者 v0.5.x-beta，本地 v0.0.x 稳定） |
| 09-01 | 9e75c6b 修复目录整理误匹配 | mediaorganize/tmdb | 无关 |
| 09-01 | 1c71fec 账号连接检测优化 | account ping | 中性，本地账号体系已改动大，需人工比对 |
| 09-01 | dd4c13d 光鸭改为本地扫码 | Guangya | 无关（该驱动本地已删） |
| 08-30 | a1162dc 修复手动匹配 | mediaorganize binding | 无关 |
| 08-30 | b73e6c2 115open离线支持ed2k | offline | 无关（离线已删） |
| 08-30 | de83b46 上传任务批次化 | upload/local_upload 任务树 | **相关**：含 local_upload.go、upload manager 改动，本地保留上传链路，但改动大（migration 0022 + SSE + 任务树），需评估后决定 |

源码级 diff（排除 gz/js/css/map）：171 files，+7893/-1472，大头是跨盘/STRM/上传任务树/认证重构。

## 同步建议

- **不 merge、不 rebase**：方向已分叉，整体同步会撤销全部精简成果。
- 值得人工评估的 3 个（按优先级）：
  1. `353b830`（目录错位，8 行小补丁，风险低，建议优先看）；
  2. `c7a424c` 中的 `drivers/115_Open/auth.go`、`drivers/189Cloud/*` 部分（认证修复，需逐文件比对，本地 auth 链路已被精简包围，勿整提交合并）；
  3. `de83b46`（上传批次化，改动大，需先在测试环境验证 DB migration 0022 与 SSE/任务树前端是否与本地 UploadTaskSettingsPanel 精简冲突）。
- 其余 17 个建议忽略。
- 版本线说明：作者走 `v0.5.x-beta`，本地走 `v0.0.x` 稳定递增，两边 tag 不互通，`v0.5.4-beta` 不适用于本地。

## 只读确认

全程只执行了 `git fetch origin` + `git log/show/diff`，未切换分支、未合并、未修改工作区文件（除本任务目录外 `git status` 干净）。
