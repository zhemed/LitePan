# Implementation Plan: 部署 LitePan v0.0.44 到本机 :5211

## Overview

顺序：**前置快照 → up → 可用性验证 → 持久化验证（含 restart）→ 边界核对 → 交付说明 → 归档**。

本任务**不产生代码改动**（除任务目录），故质量门聚焦"部署契约"而非 lint/test：真正要证的是**服务可用**与**数据可持久**两件事。

本机 `danger-full-access` 且审批关闭 —— **不请求任何 escalation**。

---

## Phase 0: 前置快照（回滚锚点）

- [ ] 0.1 `git status --short` 记录基线（期望：仅本任务目录）
- [ ] 0.2 记录端口基线：`ss -ltnp | grep -E ':(5211|42069|3080|3081)\b'`（期望 5211/42069 空闲；3080/3081 由 node 持有，记下 PID 供后比对）
- [ ] 0.3 记录 Docker 基线：`docker ps -a`（期望无 litepan）、`docker images ghcr.io/zhemed/litepan`
- [ ] 0.4 确认 `data/`、`mounts/` **不存在**（证明首次启动只新建、不覆盖）
- [ ] 0.5 记录改密前凭据状态：确认这是全新库（无既有 `data/`），故首次登录应为默认口令 + `must_change_password:true`

**回滚点 R0**：以上全部写入本任务记录；DSH 进程 PID 已记

---

## Phase 1: D1 起容器

- [ ] 1.1 若存在同名容器先清干净（**不动 data/**）：`docker compose down --remove-orphans || true`
- [ ] 1.2 `cd /root/LitePan && docker compose up -d`
- [ ] 1.3 **验证门 G1**：
      - `docker compose ps` → `litepan` 状态 `Up`
      - `docker compose ps --format '{{.Status}}'` → 含 `Up`
      - 等待启动：轮询 `/api/health` 至 200（最多 ~30s，启动含认证调度退避）
- **回滚点 R1**：`docker compose down`（保留 data/）

---

## Phase 2: D2 可用性验证

- [ ] 2.1 `ss -ltnp | grep 5211` → 有监听
- [ ] 2.2 `curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:5211/api/health` → `200`；响应体含 `"status":"ok"`
- [ ] 2.3 `curl -s http://127.0.0.1:5211/` → 含 `LitePan` 的 SPA
- [ ] 2.4 表单编码登录（**不得发 JSON**）：`curl -s -X POST http://127.0.0.1:5211/api/auth/login -d 'username=admin&password=admin' -c /tmp/deploy-cookie`
      → 200 且 `is_admin:true`、`must_change_password:true`
- [ ] 2.5 带 session 取版本：`curl -s -b /tmp/deploy-cookie http://127.0.0.1:5211/api/public/system-config`
      → `version == "v0.0.44"` 且既有 3 字段均在
- [ ] 2.6 `docker logs litepan 2>&1 | grep -q "HTTP 服务已监听"`
- **验证门 G2**：2.2–2.6 全过

---

## Phase 3: D3 持久化验证（本任务核心）

- [ ] 3.1 宿主机侧：`ls -l /root/LitePan/data/litepan.db`（存在，记下 size/inode）
- [ ] 3.2 容器侧：`docker compose exec -T litepan ls -l /app/data/litepan.db`（记下 size/inode）
- [ ] 3.3 **同一 bind mount 证明**：两侧 `stat` 的 **inode 相同**（`docker compose exec -T litepan stat -c '%i %s' /app/data/litepan.db` vs 宿主机同命令）
- [ ] 3.4 记录重启前证据：DB size、`data/` 内文件清单
- [ ] 3.5 `docker compose restart` → 等 `/api/health` 回到 200
- [ ] 3.6 **验证门 G3（数据未重置）**：
      - 重启后 DB size **未变**（或按正常写入合理增长，**不得显著变小**）
      - 用重启前拿到的 session cookie 复测 `/api/public/system-config` **仍 200**（证明会话/库未被重置）
      - `data/` 内文件清单与重启前一致（无重建痕迹）
- [ ] 3.7 restart 策略核对：`docker compose ps --format '{{.Name}} {{.Status}}'` 与 `docker inspect litepan --format '{{.HostConfig.RestartPolicy.Name}}'` → `unless-stopped`
- **回滚点 R3**：`docker compose down`（**数据保留**，可再次 up 复现）

---

## Phase 4: D5 边界核对

- [ ] 4.1 DSH 未受影响：`ss -ltnp | grep -E ':(3080|3081)\b'` 仍在监听，且 PID 与 Phase 0 记录一致
- [ ] 4.2 仓库未被污染：`git status --short` → 仅本任务目录；`data/`、`mounts/` **不出现在状态中**（gitignored 生效）
- [ ] 4.3 `docker compose config` 能正常解析（compose 文件本身有效）
- [ ] 4.4 **验证门 G4**：4.1–4.3 全过

---

## Phase 5: 交付说明

- [ ] 5.1 记录入口 URL（本机 IP + 5211）与默认口令处置提示（**用户自行改密**）
- [ ] 5.2 记录数据位置与备份要点：`/root/LitePan/data/litepan.db` 与 `secret.key`（备份时两者一起）
- [ ] 5.3 记录回滚方式（`docker compose down` 保留数据）
- [ ] 5.4 清理临时文件：`rm -f /tmp/deploy-cookie`

---

## Phase 6: 归档收尾

- [ ] 6.1 `skill trellis-check` 走查（本任务无代码改动 → 质量门落在部署契约与边界核对）
- [ ] 6.2 `flow_gate.py mark-check 09-12-deploy-litepan-local-5211 --note "<部署+持久化摘要>"`
- [ ] 6.3 勾选 `prd.md` 全部验收项
- [ ] 6.4 `flow_gate.py pre-archive` → `task.py archive ... --skip-branch-validation`
- [ ] 6.5 `add_session.py`（会话号以 journal 为准）→ 手工 commit（若 archive auto-commit 未覆盖）→ `git push`

---

## Validation Commands（汇总）

```bash
cd /root/LitePan
docker compose ps
ss -ltnp | grep -E ':(5211|3080|3081)\b'
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:5211/api/health
curl -s http://127.0.0.1:5211/ | head -c 200
curl -s -X POST http://127.0.0.1:5211/api/auth/login -d 'username=admin&password=admin' -c /tmp/deploy-cookie
curl -s -b /tmp/deploy-cookie http://127.0.0.1:5211/api/public/system-config
stat -c '%i %s' /root/LitePan/data/litepan.db
docker compose exec -T litepan stat -c '%i %s' /app/data/litepan.db
docker inspect litepan --format '{{.HostConfig.RestartPolicy.Name}}'
docker logs litepan 2>&1 | tail -20
git status --short
```

## Review Gates

| 门 | 位置 | 判据 |
|---|---|---|
| G1 | Phase 1.3 | 容器 Up + health 可达 |
| G2 | Phase 2.6 | health/SPA/登录/版本/日志五连全过 |
| G3 | Phase 3.6 | 内外 inode 相同 + restart 后数据与会话均未丢 |
| G4 | Phase 4.4 | DSH 监听与 PID 未变 + 仓库无受跟踪改动 |

## Rollback

见 design.md → **Rollout / Rollback**。核心：**只 `docker compose down`，绝不删 `data/`**；本任务不会覆盖任何既有数据（部署前已确认 `data/` 不存在）。

## Out of Scope

- 修改 `docker-compose.yml`（含加 healthcheck、改端口、改 restart 策略）
- 修改管理员口令（用户自行在界面完成）
- 配置反向代理 / TLS / 域名
- 创建网盘账号、挂载、FUSE 实际挂载测试
- `docker-compose.fnos.yml` 的过期 tag（`v0.0.31`）—— 属另一议题，已在 PRD Notes 记录
