# 09-12-investigate-dead-code-sweep

## Goal

**全面排查本方精简分支残留的死代码**，覆盖 Go 后端（无消费者包、不可达符号、遗留表/列、注册表残留）、前端（无引用文件、未用依赖）、构建产物与配置项；对每一项给出**可复核的证据**（避免"文件存在即在用"的旧判据误判），并输出**按优先级排序的清理建议清单**（含规模与风险），供后续清理任务使用。本轮**只读**，不实施任何删除。

## Background

- 触发：`09-12-investigate-mediaorganize-necessity` 查实 `internal/mediaorganize/rules` 为孤儿包（零 import、不在 `cmd/litepan` 依赖图、已死 13 天），随后由 `09-12-remove-mediaorganize-dead-package` 删除（0.0.40）。用户要求以此为例**全面排查**是否还有同类残留。
- 本方是上游 `Ponphil/LitePan` 的精简分支：`1bcfac8`（2026-08-30）与后续提交删除了大量模块（STRM / 跨盘 / 缓存任务 / 目录整理 / 分类 / 清理 / 海报 / Emby 反代 / 多驱动 / AI / 公告 等），历史删除**可能留下孤儿**（包、文件、依赖、表、路由、配置、前端组件）。
- 已知判据教训：**"文件存在"≠"代码可达"**；Go 侧可用 `go list -deps ./cmd/litepan` 与 `deadcode` 工具一票否决，前端需引用计数 + 人工复核。

## Requirements

- **R1 只读**：不删除、不修改任何代码/配置/DB；允许安装/运行只读分析工具（如 `golang.org/x/tools/cmd/deadcode`）。
- **R2 覆盖范围（六类）**：
  1. **Go 包级**：仓库内所有包 vs `cmd/litepan` 依赖图 → 找出无消费者包；
  2. **Go 符号级**：`deadcode` → 生产不可达的函数/方法，以及仅被测试使用的代码；
  3. **API/注册表残留**：路由、wire、driver registry、自动化动作、设置项枚举中指向已删功能的条目；
  4. **DB 遗留**：迁移脚本 / store 层中属于已删模块的表、列、索引、清理逻辑（只读检查，可读本地 `data/litepan.db` 结构，不写）；
  5. **前端**：`web/src/**` 中无外部引用的组件/composable/util/样式/api 模块；`web/package.json` 中未使用的依赖；
  6. **构建产物/资源**：`internal/api/web/assets` 中不被任何入口引用的旧 chunk、未被引用的静态资源。
- **R3 证据与防误报**：每项发现必须附**可复核证据**（命令 + 输出摘要），并做可达性复核（动态引用、反射、注册表、字符串常量、测试独占使用等都要在结论中说明），不得仅凭"名字看起来像"下结论；无法确认的标为"待定"。
- **R4 输出**：`research.md` 含——分类清单（每项：路径/规模/证据/可达性结论/建议动作）、误报排除记录、按优先级排序的清理建议（规模 + 风险 + 依赖关系）、不建议清理的项及理由。
- **R5 记录**：本轮零代码改动（`git status` 除任务目录外为空）；结论写入 PRD 勾选项与 journal。

## Constraints

- 不触碰生产机；不改本地容器与数据；不跑任何写库语句（DB 只读打开）。
- 不扩张为"顺手删"：**所有清理留给后续任务**。
- 分析工具安装仅限 `$GOPATH/bin`（不写入仓库）。
- 时间盒：聚焦"可判定的死代码"，模糊项标注"待定"而非穷举。

## Acceptance Criteria

- [x] 已完成 Go **包级**排查：列出 `cmd/litepan` 依赖图之外的全部仓库内包（含证据命令）
      → research.md §3：45 个仓库内包中 4 个不在生产依赖图（`internal/proxybase`、`internal/taskauth`、`pkg/strutil` 为真死；`drivers/template` 为测试专用脚手架）
- [x] 已完成 Go **符号级**排查：至少用 `deadcode` 跑一轮并给出结论摘要（若工具不可用，说明原因并给替代证据）
      → research.md §4：`deadcode ./cmd/litepan` 32 个生产不可达 → 与 `deadcode -test ./...` 做差集得 **24 个真死 / 8 个仅测试可达**；并逐项核对归属的已删功能
- [x] 已完成 API/注册表/自动化动作/设置枚举的残留排查
      → research.md §6：路由无已删功能残留、驱动注册表仅 3 驱动、自动化动作仅 `local_upload`、设置键全部在用；另发现 1 个疑似未使用端点 `/accounts/{id}/refresh-auth`
- [x] 已完成 DB 遗留表/列排查（只读；含本地库结构核对）
      → research.md §7：本地库 11 张表全在用、已删功能表由迁移 0022 清除；迁移历史 9 文件建议保留；`upload_tasks` 跨盘遗留列仍被本机上传路径使用
- [x] 已完成前端无引用文件与未用依赖排查（含引用计数方法说明）
      → research.md §5：223 个源文件中 **12 个零引用**（2,237 行，含全仓二次复核）；30 个依赖 **0 未用**
- [x] 已完成构建产物残留排查
      → research.md §8：109 个 asset（102 gz + 7 不可压缩）**全部被引用**，0 残留
- [x] `research.md` 已产出：分类清单 + 每项证据 + 可达性结论 + 误报排除记录
      → 已产出，含 §10 误报排除（HashMD5 同名常量、template 非死包、面板未接线≠功能已删、测试接口桩不计）
- [x] 已给出按优先级的清理建议清单（规模/风险/依赖），并列出"不建议清理"项及理由
      → research.md §9：P1 死包 + P1 死函数 / P2 前端 12 文件（分两步）/ P3 template+OAuth（默认保留）/ P4 预留端点；不建议清理：迁移历史、跨盘列
- [x] 本轮零代码改动（除任务目录与 journal）
      → `git status` 仅任务目录；未改任何代码/配置/DB（DB 以 `mode=ro` 打开）

## Notes

- 本任务为调查类（`scope=lightweight`），无代码产出；清理需另建任务（建议 P1 两项合并为一个纯删除任务）。
- **方法沉淀（供后续复用）**：Go 侧「`go list -deps` 包级 + `deadcode` 符号级（prod 与 `-test` 差集）」；前端「basename/stem 引用计数 + 全仓二次检索」；产物「解压 .gz 后互引检查」；路由「叶子路径 × 前端文本」。凡 grep 与 deadcode 冲突，以调用链是否可达为准。
- **已知脚本事故（如实记录）**：路由使用率首版脚本按缩进重建 Route 前缀出错（把 admin 路由拼成 `/api/public/admin/...`），结果作废；改用叶子路径匹配后仅剩 1 条未命中。该"端点使用率"核对仍标注为**待定**（动态拼接路径无法穷尽）。
- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
