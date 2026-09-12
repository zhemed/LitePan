# P6 必要性调查报告（目录整理 / mediaorganize）

> 任务：`09-12-investigate-mediaorganize-necessity`｜口径：**只读**（零代码改动）
> 时间：2026-09-12｜结论：**P6 不需要移植——它要改的是本方已死的代码**

---

## 1. 结论摘要

1. **P6 不需要**：`internal/mediaorganize/rules` 在本方精简分支里是**孤儿包（死代码）**——没有任何包 import 它，也不在主程序依赖图里（`go list -deps ./cmd/litepan | grep -c mediaorganize` = **0**，即连二进制都不包含它）。上游 `20b66de` 改的正是这个包，移植它等于修改不可达代码。
2. **用户记忆正确**：目录整理功能早在 `1bcfac8 refactor(cache,organize): remove cache tasks and directory organization`（2026-08-30）就已删除；只是**该包未一并删除**，成了残留。
3. **意外发现（两处记录过期，容易误导后续判断）**：
   - journal-1.md 记载"保留 `mediaorganize/rules` 供 `file/name_align` 使用"——与实际不符：**同一个提交 `1bcfac8` 就把 `name_align.go` 的 `mrules` 依赖换成了自带简化解析**；
   - spec `.trellis/spec/backend/backend/directory-structure.md:44` 仍写"`mediaorganize/rules` 保留"，会让读者（包括上一轮做上游调查的我）误判它是活跃代码——**这正是 P6 被误列为候选的原因**。
4. 顺带澄清：**"命名对齐"功能仍然在线**（`/api/.../name-align/preview`、`NameAlignModal.vue`），只是它已不依赖 `rules` 包。所以"整理"死、"命名对齐"活，两者不要混淆。

---

## 2. 证据链（全部只读）

### 2.1 import 链：零消费者

```bash
$ grep -rn "mediaorganize" --include="*.go" . | grep -v "^./internal/mediaorganize/"
./internal/file/name_align.go:394:  // Simple extraction without mediaorganize/rules: try SxxExx, CN, etc.

$ grep -rn "ParseFilenameWithGuessit\|mrules\|\"litepan/internal/mediaorganize" --include="*.go" .
（仅 package 内部自引用 + 上面的注释）

$ GOWORK=off go list -deps ./cmd/litepan | grep -c mediaorganize
0            # 主程序依赖图中完全没有它
```

⇒ 该包只被自己的 4 个 `_test.go` 引用；`go test ./...` 仍会跑它的单测（`ok litepan/internal/mediaorganize/rules`），但**没有任何生产路径能到达**。

### 2.2 功能入口：三类入口全部不存在

| 入口 | 检索 | 结果 |
|---|---|---|
| HTTP 路由 | `grep -n "organize" internal/api/router.go` | 空 |
| 前端 UI | `grep -rln "目录整理\|media-organize\|mediaOrganize\|organizePlan" web/src` | 空 |
| 前端构建产物 | `grep -rl "目录整理" internal/api/web/assets` | 空 |
| 自动化动作 | `grep -n "AutomationAction.*=" internal/domain/automation.go` | **只剩 `AutomationActionLocalUpload`**（整理/清缓存/STRM 等动作已删） |

### 2.3 历史：什么时候变成孤儿的

```bash
$ git log --oneline -5 -- internal/mediaorganize/
1bcfac8 refactor(cache,organize): remove cache tasks and directory organization
f3bbc8c refactor(share): remove file share (WebDAV dav) completely
38a8331 refactor(strm): remove STRM feature completely
...

$ git show 1bcfac8 -- internal/file/name_align.go | head -30
-	mrules "litepan/internal/mediaorganize/rules"
...
-	parsed := mrules.ParseFilenameWithGuessit(name)
-	parsed = mrules.ApplyEpisodeFallbacks(name, parsed)
+	// Simple extraction without mediaorganize/rules: try SxxExx, CN, etc.

$ git log --oneline -S "mrules" -- internal/file/name_align.go
1bcfac8 refactor(cache,organize): remove cache tasks and directory organization
a69bb6e 新版首次提交
```

⇒ 根提交时 `name_align` 依赖 `rules`；`1bcfac8` 移除依赖并把解析改成自包含实现；`rules` 包自此无消费者（**已死 13 天**）。

### 2.4 规模

| 项 | 值 |
|---|---|
| 非测试文件 | 15 个 / **3,796 行** |
| 测试文件 | 4 个 |
| 对外部包的依赖 | 无（`grep "litepan/" internal/mediaorganize/rules/*.go` 为空，纯自包含） |

### 2.5 上游对照（P6 的出处）

```bash
$ git show --stat 20b66de            # 上游「修复目录整理误匹配」
internal/mediaorganize/rules/tmdb.go + tmdb_test.go      # ← 本方已死
drivers/115_Open/full_list_test.go、drivers/Guangya/qrlogin_test.go   # ← 本方已删驱动
$ git show origin/main:internal/file/name_align.go | grep -n mediaorganize
12:	mrules "litepan/internal/mediaorganize/rules"        # 上游 name_align 仍在用 rules
```

⇒ 上游的 `rules` 是活跃代码（目录整理功能 + name_align 都在用）；本方两者都不用。**P6 的"候选"身份来自"文件存在"这一表面判据，属误判**。

---

## 3. 更正记录

| 位置 | 原表述 | 更正为 |
|---|---|---|
| `09-12-investigate-upstream-updates` 报告 §6 P6 | "本方保留 `mediaorganize/rules`（19 文件），可减少整理误匹配" | **作废**：该包在本方无消费者、不在二进制内；P6 从候选清单移除（候选由 9 项减为 8 项） |
| 同上报告 §3.2 第 6 行 | "候选 P6（`mediaorganize/rules/tmdb.go` + 测试）" | 同上作废，改判"不适用（本方为死代码）" |
| journal-1.md:252 | "keep … `mediaorganize/rules` for file/name_align mrules.ParseFilenameWithGuessit" | 与代码不符：`name_align` 的该依赖已被同一提交移除（本次仅记录，不改历史 journal） |
| spec `directory-structure.md:44` | "…`internal/cache` 核心与 `mediaorganize/rules` 保留" | **待同步**：`rules` 实际无消费者（见 §4 建议） |

---

## 4. 建议（可选，需另建任务）

**建议 A（推荐，成本极低）：删除孤儿包并同步 spec**
- 范围：删 `internal/mediaorganize/`（15 非测试文件 + 4 测试文件，3,796 行）、更新 `.trellis/spec/backend/backend/directory-structure.md` 第 44 行与相关表格行、`concurrency-and-scheduling.md:41`（其中 `mediaorganize.Service` 一行同样已过期）。
- 收益：消除"看起来还活着"的误判源（本次 P6 误判即由此而来）；`go build ./...`/`go test ./...` 更快一点；仓库更干净。
- 风险：低。若将来想恢复"目录整理/更强文件名解析"，可从上游取回（本方走内容对照移植），或直接恢复本次删除的提交。
- 影响面核实：该包无外部依赖、无入口 → 删除不影响任何在线功能。

**建议 B（保守）：保留不动**
- 理由：不影响运行与体积（Go 链接器不会把它打进二进制）；上游若继续演进该包，本方将来若要重做目录整理会更省事。
- 代价：错误判据会再次出现（下一个人看到目录里有这个包，就会以为它还在用）。

**不做的事**：本任务不实施任何删除（如需删除，另建任务走九步流程）。

---

## 5. 未验证 / 限制

- 未验证"将来是否要重做目录整理"这一产品决策（取决于用户）。
- 未评估删除后对 `go test ./...` 耗时/仓库体积的精确量化（仅定性：该包 3,796 行 + 4 个测试文件）。
- 未触碰生产机、未改任何容器/数据。

---

### 附：本次执行的关键命令

```bash
grep -rn "mediaorganize" --include="*.go" . | grep -v "^./internal/mediaorganize/"
grep -rn "ParseFilenameWithGuessit\|mrules" --include="*.go" .
GOWORK=off go list -deps ./cmd/litepan | grep -c mediaorganize
grep -n "organize" internal/api/router.go
grep -rln "目录整理" web/src internal/api/web/assets
grep -n "AutomationAction.*=" internal/domain/automation.go
git show 1bcfac8 -- internal/file/name_align.go
git show --stat 20b66de
git show origin/main:internal/file/name_align.go | grep -n mediaorganize
```
