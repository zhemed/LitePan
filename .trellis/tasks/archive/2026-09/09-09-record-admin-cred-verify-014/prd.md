# 09-09-record-admin-cred-verify-014

## Goal

记录管理员新凭据 `admin/123456` 到 AGENTS.md（沿用既有约定），并用新凭据完成 0.0.14 天翼云盘修复的实测验收。

## 执行记录

1. **凭据核实**：DB 显示 `admin_password` 最后更新 `2026-08-30 16:38`（10 天未改），`admin/admin` 与 `admin/123456` 均登录失败 → 用户实际改密未发生在本实例。
2. **授权重置**：`ask_user_question` 用户选择"授权把本地密码重置为 123456"。用应用自身 `pkg/security.HashPassword`（pbkdf2-sha256/600000轮/16B盐，自校验通过）生成哈希；重置前 `sqlite3 .backup` 全库备份至 `/tmp/litepan-backup-20260909-201827.db`；UPDATE 后重启容器。
3. **登录仍失败 → 破案**：失败日志 `username=""`——`POST /api/auth/login`（`internal/api/auth.go:13`）只走 `r.ParseForm()`，**仅接受 form 表单编码**；此前所有 JSON 登录请求从未被解析过（包括最早的 admin/admin 尝试）。
4. **实测验收（0.0.14 修复闭环）**：form 编码登录成功（`must_change_password:false`）→ `/api/files/list?account_id=1` **success:true** 返回真实目录；日志还原恢复链：20:16:30 用户 UI 请求触发 `请求链路触发认证凭证回写`（400+payload→CodeAuthExpired→被动刷新→凭据落库 `last_refresh_at=20:16:30`）→ 20:19:14 列表直接成功。
5. **AGENTS.md**：更新凭据记录 + 新增 form 编码注意事项；清理误建空文件 `data/litean.db`（此前 sqlite3 命令笔误产生）与 `.tmp-hashgen/`。

## Acceptance Criteria

- [x] admin/123456 可登录本实例（AGENTS.md 已记录，含 form 编码注意事项）
- [x] 0.0.14 天翼云盘修复实测通过（列表成功 + 被动刷新恢复链日志证据）
- [x] 重置前 DB 全量备份落盘
- [ ] 归档 + journal + push
