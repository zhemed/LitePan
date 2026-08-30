# 调查 LitePan-own 实际在用与融合一致性

## Goal
以只读方式查明 `LitePan-own` 在 `10.0.0.99`（闭环）与预期生产 `10.0.0.11` 的**真实在用**配置（映射、自动化规则、驱动/上传参数）与 `zhemed/LitePan @ 0.0.3` 的融合实现是否对得上，输出逐项对照表与缺口。

## Background
- `LitePan-own` 已 `git -C LitePan-own log 93616d6..099cbc9` 9 commits 提取完整，`0.0.3` 已补齐 5 处高优修复，但“实际在用”指运行时配置而非代码：`docker-compose.yml` 3 映射、`settings KeyLocalUploadMappings`、`automation` 规则（daily 02:00 等）、`115_Open 600s/512MB` 是否真的在跑、`B mode` 路径是否一致。
- 工作区可取证：`LitePan-own/docker-compose.yml`、`LitePan-own/data`（若有）、`_extracted`、`LitePan/data/litepan.db`（LitePan 侧）、`git` 历史，以及 `10.0.0.99` 的 `docker ps / logs`（若可连）。
- 用户问题：`这几个自定义都是干什么的` → 进一步问真实在用 vs 融合是否对齐。

## Requirements
- **静态在用**：读取 `LitePan-own/docker-compose.yml`、`LitePan-own/README.md`、`_extracted/README_CUSTOM.md`、`LitePan-own git diff` 与 `LitePan/docker-compose.yml` 对比，明确 3 映射路径、镜像标签、 volumes。
- **规则在用**：若可读 `LitePan-own` 的 `data/litepan.db`（或 `LitePan` 的），`sqlite` 查 `automation_rules / automation_runs / settings` 的真实规则（触发器、动作 `local_upload` 的 `mappings/account/target`），否则以 `README_CUSTOM.md` 与 `AutomationPanel.vue` 注释为准并标注“未能独立验证”。
- **驱动在用**：核对 `115_Open driver 600s / upload 512MB`、`189Cloud batchTaskInfos`、`file Delete` 是否在 `LitePan` 侧 `0.0.3` 已等价，且与 `LitePan-own` 实际跑的镜像 `ghcr.io/zhemed/litepan-own:0.0.8/0.0.1` 一致。
- **产出**：`investigation.md` 含 ① 实际在用清单（映射、自动化、驱动）② 与 `LitePan 0.0.3` 融合对照（逐行 ✅/❌）③ 结论与建议。

## Constraints
- 只读，不改 `LitePan-own/`、`_extracted/`、业务代码；写仅限任务目录 `investigation.md`。
- 若 `10.0.0.99` 不可达或 `data/litepan.db` 无权限，明确标注“未能独立验证”并以 `git/README` 为准。
- 遵循 `AGENTS.md 项目强制规则`：`task.py start` 后执行。

## Acceptance Criteria
- [ ] `investigation.md` 存在，含静态在用、规则在用、驱动在用三表
- [ ] 每项标注来源（`git:xxx` / `file:xxx` / `db:xxx`）与验证状态（已验证/未能独立验证）
- [ ] 明确结论：对得上 / 哪几项对不上（文件:行）
