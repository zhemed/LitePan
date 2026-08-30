# Remove docker pull and git clone from README

## Goal

按用户“不要不要，这两个去掉”将 `README.md` 的 `快速开始` 中 `docker pull ghcr.io/zhemed/litepan:v0.5.2-Beta` 与 `git clone https://github.com/zhemed/LitePan.git && cd LitePan && docker compose up -d\n# http://IP:5211  admin / admin` 两段彻底删除，仅保留 `services: litepan: image: ghcr.io/...` 的 `compose` 块。

## Requirements

- **删除段落 1**：`docker pull ghcr.io/zhemed/litepan:v0.5.2-Beta` 的 `bash` 代码块（` ```bash\ndocker pull ...\n``` `）
- **删除段落 2**：`git clone https://github.com/zhemed/LitePan.git && cd LitePan && docker compose up -d` 与 `# http://IP:5211  admin / admin` 的 `bash` 代码块
- **保留**：`## 快速开始` 标题、` ```yaml services: litepan: image: ghcr.io...` 的 `compose` 块、`支持网盘` 等其余 4 屏

## Constraints

- 仅删 `README.md` 的上述 2 个 `bash` 块，不改 `compose` 的 `yaml` 块与 `internal/*` 代码
- `grep -c "docker pull ghcr.io/zhemed/litepan:v0.5.2-Beta" README.md` == 0
- `grep -c "git clone https://github.com/zhemed/LitePan.git" README.md` == 0
- `grep -c "http://IP:5211  admin" README.md` == 0

## Acceptance Criteria

- [ ] `README.md` 无 `docker pull ghcr.io/zhemed/litepan:v0.5.2-Beta` 的 `bash` 块
- [ ] `README.md` 无 `git clone https://github.com/zhemed/LitePan.git` 的 `bash` 块
- [ ] `README.md` 仍含 `services: litepan: image: ghcr.io/...` 的 `yaml` 块
- [ ] `wc -l README.md` 约 45-55（原 63 行删 2 段后）
- [ ] `GOWORK=off go vet ./...` 仍 PASS

## Notes

- 用户明确“不要不要，这两个去掉”，为轻量文档任务
