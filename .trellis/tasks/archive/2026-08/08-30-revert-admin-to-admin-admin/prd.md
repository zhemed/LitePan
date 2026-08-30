# Revert admin to admin/admin

## Goal

将线上管理员凭据从 `admin / 123456`（`2026-08-30` 设）改回默认 `admin / admin`（`must_change:true`，首次登录需改密），符合用户最新指令。

## Requirements

- **凭据**：`admin_username=admin`，`admin_password` 为 `admin` 的 `pbkdf2:sha256:600000$...` 哈希（`security.HashPassword("admin")`），`admin_session_generation` 清空，`admin_temp_password_hash` 清空
- **落盘**：`data/litepan.db` 的 `configs` 表 `admin_password` 更新，`admin_temp_password_expires_at=0`
- **容器**：`litepan` 容器重启后生效（`litepan-go:three-drivers` 镜像不变，仅 `data` 变）
- **文档**：`README.md` 的 `快速开始` 中 `默认管理员 admin / 123456` 改回 `admin / admin`（与 `AGENTS.md` 的强制规则同步）

## Constraints

- 仅改 `data/litepan.db` 的 `configs` 与 `README.md` 的一行，不改 `drivers`/`internal` 代码
- 操作前 `cp data/litepan.db data/litepan.db.bak.$(date +%s)` 备份
- 改后 `POST /api/auth/login -d "username=admin&password=admin" → 200 must_change:true` 且 `123456 → 401`

## Acceptance Criteria

- [ ] `sqlite3 data/litepan.db "SELECT substr(value,1,20) FROM configs WHERE key='admin_password'"` 的 `VerifyAdminPassword(..., "admin")==true` 且 `Verify("123456")==false`
- [ ] `curl -s http://127.0.0.1:5211/api/auth/login -d "username=admin&password=admin" -c /tmp/c.txt -i | grep 200` 且 `curl -b /tmp/c.txt /api/auth/status | grep must_change.*true`
- [ ] `curl -s http://127.0.0.1:5211/api/auth/login -d "username=admin&password=123456" | grep 401`
- [ ] `grep -c "admin / 123456" README.md` == 0 且 `grep -c "admin / admin" README.md` >=1
- [ ] `grep -c "admin / 123456" AGENTS.md` == 0 且 `grep -c "admin / admin" AGENTS.md` >=1
- [ ] `docker logs litepan --tail 20 | grep -i panic` == 0，`/api/health 200`

## Notes

- 本任务为轻量（仅数据+文档），`design.md/implement.md` 可省略，`prd` 即验收
