# 09-09-investigate-upstream-merge-safety

## Goal

只读调查：判断上游 `origin`（Ponphil/LitePan，现 `374affd`，merge-base `4c160d9` 之后 20 个新提交）能否安全合并进本地精简分叉（`main` = `b77c1f7`，merge-base 之后本方 178 提交）。产出**合并可行性结论 + 冲突清单 + 建议方案**。本次**不执行合并**，不改动业务代码。

## Requirements

1. 明确上游 20 个新提交的主题分类（哪些属于已删除功能、哪些与本方保留功能相关）。
2. 用 `git merge-tree`（无需改工作区）模拟合并，统计真实冲突文件数与逐文件冲突原因。
3. 评估上游变更与本方精简面的重叠：STRM、跨盘传输(cross_transfer)、整理(organize)、联动(webhook/通知)、emby、驱动(115/189/local) 相关提交各自落在哪些文件、这些文件在本方是否已删除或重写。
4. 对保留功能（115_Open/189Cloud 驱动、本地上传、automation daily/interval+local_upload、认证刷新）标记上游是否有值得吸收的修复（如认证问题修复 c7a424c/8e332f3、上传目录错位 353b830、115 open ed2k b73e6c2）。
5. 结论三选一：A) 可直接 merge；B) 需 cherry-pick 少数提交；C) 不建议合并，列出可单独移植的修复点。

## Constraints

- 只读调查：`git fetch/merge-tree/diff/show` 允许；禁止 `git merge/rebase/cherry-pick/checkout` 改分支、禁止改业务代码、禁止 docker。
- 工作区 `main` 干净（`b77c1f7`），调查结束后保持干净。
- 版本号不变更（纯调查，无发布）。

## Acceptance Criteria

- [x] 上游 20 提交逐条分类表（主题/涉及文件/与本方删除面是否重叠）→ research.md §2
- [x] `git merge-tree` 模拟结果：68 个真实冲突文件（50 delete/modify + 18 content）→ research.md §3
- [x] 保留功能相关的可移植修复点清单（提交号 + 文件 + 风险）→ research.md §2.2/§4
- [x] 结论与建议方案（A/B/C）→ research.md §1/§4：**C 不建议合并，改按需 cherry-pick**
- [ ] 归档任务并记录 journal。

## Notes

- 上游 tag `v0.5.4-beta` 已 fetch 到本地 `origin/main=374affd`。
- 参考：本方 0.0.12 分叉删除面：STRM、share/WebDAV、cache retention/organize、announcement、aux、crosstransfer、offline_download、builtin_offline、cross_transfer、webhook 触发；驱动收敛 115_Open/189Cloud/LocalFs；automation 重写为 daily/interval+local_upload。
