# 报告：GitHub与本地差距

**任务**：`09-01-investigate-github-local-gap`  
**时间**：2026-09-01 15:55 UTC  
**基线**：本地 `1a77f58` vs 远端 `github/main 1a77f58` `v0.0.10 9004274`  
**方法**：`git log/status/ls-remote + gh api + sha256 + docker` 只读

---

## 一、差距总览

| 维度 | 本地 | 远端 | 差距 | 结论 |
|---|---|---|---|---|
| **HEAD** | `1a77f58 chore(task): archive 09-01-rollback-to-65b868b` | `1a77f58` 同 | `git log HEAD..github/main 0` `github/main..HEAD 0` | ✅ 一致 |
| **分支** | `main` | `main` `HEAD 1a77f58` | `git ls-remote HEAD 1a77f58` | ✅ 一致 |
| **Tags** | `v0.0.1 452b0c9` `v0.0.2 6ef108b` `v0.0.3 4d8e868` `v0.0.4 54d82dc` `v0.0.5 7439a18` `v0.0.6 feea7b0` `v0.0.7 a576484` `v0.0.8 912b97d` `v0.0.9 41cc6c6` `v0.0.10 9004274` | 同 `ls-remote` 10 tags | `git tag --list | wc -l 10` | ✅ 一致 |
| **文件** | `README 0a577f6 v0.0.10` `docker-compose v0.0.10` `install-docker.sh 105行` | `raw main` 同 `sha256 00edf3/c5e802/f3ac43` | `curl raw | sha256` 本地同 | ✅ 一致（0.0.10 已同步） |
| **镜像** | `ghcr 0.0.10/v0.0.10/latest 2026-08-31T15:?? sha256:8a405ec` `105MB` | `public` | `docker pull v0.0.10 up to date` | ✅ 一致 |
| **工作区 Clean** | `?? 2` | - | `?? .trellis/tasks/09-01-investigate-github-local-gap/`（本任务，预期） + `?? .trellis/tasks/09-01-remove-cross-disk-download/`（旧任务残留，已归档 65b868b，应 `rm -rf`） | ⚠️ 1 个残留需清理 |

**总体**：**0 差距**（`reset --hard` 后 `HEAD` 已与 `github/main` 完全同步，`0.0.10` 已发布）。

---

## 二、待清理

- ` .trellis/tasks/09-01-remove-cross-disk-download/` — 已于 `65b868b` 归档，但 `reset` 后本地又出现 `untracked` 复本（`planning` 态），应 `rm -rf` 该目录（不影响 `archive/09-01-remove-cross-disk-download`）。

---

## 三、建议

- 执行 `rm -rf .trellis/tasks/09-01-remove-cross-disk-download/` 后 `git status` 将仅剩本任务 `??`，符合 `工作区只有LitePan`。
- 后续 `0.0.11` 递增（`fix→0.0.11`）。

---

## 附：取证

```bash
git log --oneline -5; git log --oneline github/main -5
git ls-remote https://github.com/zhemed/LitePan.git | grep HEAD
gh release list --repo zhemed/LitePan --limit 5
gh api users/zhemed/packages/container/litepan/versions --jq '.[].metadata.container.tags|select(contains(["0.0.10"]))'
git status --porcelain
```
