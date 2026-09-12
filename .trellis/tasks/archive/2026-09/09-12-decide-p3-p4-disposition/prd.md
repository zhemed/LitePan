# 09-12-decide-p3-p4-disposition

## Goal

裁定死代码排查报告（`09-12-investigate-dead-code-sweep`）遗留的 **P3**（`drivers/template` + httpx OAuth 测试专用链路）与 **P4**（`POST /accounts/{id}/refresh-auth` 预留端点）是否有必要修改，还是直接关闭归档；给出可复核证据与明确结论，留档后归档。本轮**零代码改动**。

## Background

- 排查报告 §9 的 P1（3 死包 + 24 死函数）已由 `09-12-remove-dead-code-p1`（0.0.41）完成，P2（前端 23 文件 / -3,017 行）已由 `09-12-remove-dead-frontend-p2`（0.0.42）完成。
- **P3 事实**：`drivers/template` 仅被 `internal/auth/oauth_integration_test.go:14` 空导入（该测试在其注释中说明：原 123/Baidu/OneDrive 驱动已移除，改用 template 骨架驱动验证统一认证守卫）；spec `driver-development.md:71` 明确把它当作新驱动骨架（`cp -r drivers/template drivers/FooCloud`），`directory-structure.md:13` 亦列为保留驱动；httpx OAuth 辅助（`DoJSON`/`ParseDataEnvelope`/`PostOAuthProxyJSON`/`OAuthProxyHTTPError`/`OAuthProxyResponseError`）由 template 与 httpx 测试使用，生产不可达但不会被链接进二进制（0.0.40 已实测：删除死包后二进制逐字节相同）。
- **P4 事实**：`/accounts/{id}/refresh-auth` → `internal/api/auth_refresh.go`（30 行），全仓仅 `router.go:180` 一处引用；前端账号动作只有 `refresh-profile`（`web/src/api/accounts.ts:12`），无任何前端/测试/文档/脚本调用；行为＝强制对该账号做一次被动认证刷新。

## Requirements

- **R1 只读取证**：仅用 `grep`/`git grep` 与文件阅读取证，不改代码。
- **R2 结论**：对 P3、P4 各给出"保留 / 删除 / 修改"的明确结论，并列明依据与代价-收益。
- **R3 判据一致性**：结论须与既有判据一致——"生产不可达但被测试/文档依赖"不等于可删；"无调用方的公开端点"删除即改变 HTTP 契约。
- **R4 留档**：把裁定写入任务文档（`research.md`）并记录到 journal；本次即关闭排查报告 §9 的 P3/P4。

## Constraints

- 零代码改动；不触碰生产机；不改容器/DB。
- 若结论为"删除"，须另建任务（本任务不实施删除）。

## Acceptance Criteria

- [x] P3 已给出结论与证据（含"删除会失去什么"的说明）
      → research.md §1：结论 **保留**；证据＝`internal/auth/oauth_integration_test.go:14` 用它作为统一认证守卫的驱动矩阵替身（注释说明 123/Baidu/OneDrive 已移除）、spec `driver-development.md:71` 指定其为新驱动骨架；删除将削弱该集成测试覆盖并移除脚手架，而保留成本≈0（生产不可达且不进二进制，0.0.40 已实测）
- [x] P4 已给出结论与证据（含"保留成本 vs 删除代价"的权衡）
      → research.md §2：结论 **保留**；证据＝全仓仅 `router.go:180` 引用、handler 30 行、前端仅有 `refresh-profile`（`web/src/api/accounts.ts:12`）、无文档/脚本引用；删除＝改变已公开 HTTP 契约，收益仅 30 行
- [x] 两项结论与既有判据一致（测试/文档依赖、HTTP 契约）
      → 判据沿用本次排查：可达性 + 依赖价值 + 契约成本；"被测试/文档依赖"不算可删，"无调用方的公开端点"非必要不删
- [x] `research.md` 已产出并含可复现命令；本轮零代码改动（`git status` 仅任务目录）
      → research.md 已产出（含 grep 证据与命令）；`git status --porcelain` 仅任务目录
- [x] 结论已写入 journal，排查报告 §9 的 P3/P4 视为关闭
      → Session 130 记录该裁定；报告 §9 的 P3/P4 由此关闭

## Notes

- `scope=lightweight`，无代码产出。
- **裁定要点**：P3 与 P4 **均不修改**——两者都不是"遗留死代码"，而是"有明确用途的低成本资产"（测试脚手架/驱动模板、运维预留端点）。真正的死代码（孤儿包、零引用符号、零引用前端文件）已在 P1/P2 清理完毕。
- 若将来确定不再新增驱动（P3）或长期无人使用该端点（P4），可另建任务做契约级清理；届时按"有意变更"处理并在 spec 同步。
- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
