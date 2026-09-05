# 验证0.0.14认证与0.0.15文件夹上传链

## Goal

用用户提供的管理员口令登录本地5211，按两份执行报告的待验证清单逐项确认，测试文件事后清理

## Requirements

1. 用用户本次提供的管理员口令登录 `http://127.0.0.1:5211`（API），全程只读优先；口令不得写入任何文件、报告、commit。
2. 0.0.14 认证验证：账号列表状态正常；按清单触发 115 与天翼账号认证检查/刷新，确认成功且无"重新扫码"误报；不断开、不删号、不改凭据。
3. 0.0.15 上传链验证：在服务器本地建隔离测试目录（`并发2层小文件`），经 `local_upload` 接口上传到网盘侧专用测试路径（含首层目录 batch 命名），核对任务 `batch_id/batch_name/rel_path` 落库；验证后删除网盘测试路径与本地测试目录。
4. 只动 `__verify_0_0_15__` 前缀的测试路径；用户既有文件零触碰；异常即停并报告。

## Acceptance Criteria

- [ ] 管理员登录成功，账号（115/天翼）状态均有效
- [ ] 认证刷新触发成功，无误报
- [ ] 文件夹上传任务 batch 字段落库正确，网盘目录结构一致
- [ ] 测试路径（网盘+本地）已清理，无残留
- [ ] 口令未落入任何文件/commit

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
