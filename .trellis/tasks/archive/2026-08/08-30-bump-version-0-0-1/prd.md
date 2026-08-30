# Bump version to 0.0.1 stable

## Goal

将当前 `main`（`b94f91a` 的 `local_upload` 适配 + `118M 3驱动`）定为 `0.0.1` 稳定版，作为后续 `0.0.2` 递增的干净基线，使 `ghcr.io/zhemed/litepan:0.0.1` 与 `README` 的 `version-shield` 一致，已部署的 `10.0.0.11`/`99` 可通过 `0.0.2` 感知更新。

## Requirements

- **版本**：`0.0.1` 为首个稳定版，后续 `fix` → `0.0.2`，`feat` → `0.0.3`，以此类推
- **README**：`version-shield` `v0.5.2-Beta` → `v0.0.1`，`image:` 2 处 `ghcr.io/zhemed/litepan:v0.5.2-Beta` → `ghcr.io/zhemed/litepan:0.0.1`（`latest` 仍同步）
- **Docker**：`docker build -t ghcr.io/zhemed/litepan:0.0.1 -t ghcr.io/zhemed/litepan:latest .` → `docker push` 两 tag，`digest` 与 `118M` 一致
- **Git**：`git tag v0.0.1 && git push github v0.0.1`，`gh release create v0.0.1 --notes "0.0.1 稳定版：3 驱动 118M"`（可选）

## Constraints

- 仅改 `README.md` 的 `version-shield` 与 `image:` 2 处，不改 `internal/*` 代码（`buildinfo.Version` 若为 `v0.5.2-Beta` 则保持，不强求 `ldflags`）
- `README.md` 的 `grep -c "v0.0.1" >=2` 且 `grep -c "v0.5.2-Beta" ==0`（除 `WARNING` 外）
- `ghcr.io` 的 `0.0.1` 需 `public` 且 `docker pull` 可匿名拉取

## Acceptance Criteria

- [ ] `README.md` 含 `v0.0.1` 且 `image: ghcr.io/zhemed/litepan:0.0.1`
- [ ] `docker images ghcr.io/zhemed/litepan --format "{{.Tag}}" | grep 0.0.1` 有输出
- [ ] `gh api repos/zhemed/LitePan/commits/main --jq .sha` == `git rev-parse HEAD` 且 `git tag --list | grep v0.0.1`
- [ ] `curl -s http://127.0.0.1:5211/api/health | grep ok` 仍 `200`

## Notes

- 本任务为轻量，仅 `README` 与 `ghcr` 标签，不涉及 `drivers` 等已删功能
