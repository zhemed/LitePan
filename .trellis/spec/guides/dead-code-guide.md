# Dead Code Analysis Guide

> **Purpose**: find dead code **without false positives** (deleting something still reachable) and **without false negatives** (leaving dead code behind).
> Evidence base: the `09-12-*` dead-code sweeps on this repo. All numbers and examples below were measured, not assumed.

## When to Read

- Sweeping the repo for dead code (Go or frontend)
- **Before deleting** anything a tool flagged
- Investigating "is this still used?"

---

## Rule 1: Use BOTH tools — their results do not overlap

| Tool | View | Finds |
|---|---|---|
| `deadcode ./cmd/litepan` | reachability from `main` (RTA, includes interface dispatch) | production-unreachable functions |
| `golangci-lint run --enable=unused -c .golangci.yml ./...` | per-package, **includes `_test.go`** (U1000) | unused identifiers inside packages |

Measured 2026-09-12: `v0.0.44` → `deadcode` **9**, `unused` **14**；两轮清理后 `v0.0.45`（本轮发版）→ `deadcode` **7**, `unused` **0**，**两个视角始终零重叠**。

- `unused` treats a package's own tests as *users* → it is blind to production symbols that only tests reach
- `deadcode` never looks inside `_test.go` → it is blind to dead test code

**Running only one guarantees a blind spot.** The previous sweep used `deadcode` alone and therefore never saw the 14 dead test-layer symbols.

---

## Rule 2: Before concluding, run two hard checks

### (a) Interface-implementing methods → check instantiation FIRST

A method that *looks* like an interface stub is only justified if **something instantiates the type and assigns it to an interface variable**. "Looks like a mock" is not evidence.

```bash
# 判定手法：搜类型名，看有没有 `&Type{` 形式的实例化（只有 type/方法定义 = 真死）
grep -rn "TypeName" internal/<pkg>/
```
历史案例（目标文件已删，手法照旧有效）：`internal/settings/service_test.go` 的 `memoryConfigRepo`、`internal/automation/service_test.go` 的 `apiKeyRepo` —— 都只有定义与方法、无任何实例化，已确认删除。

Both an earlier sweep and an early draft of a later one misclassified such a batch as "interface stubs, not dead code". Checking instantiation is what settles it.

**Counter-example (correctly kept)**: `pkg/jsonvalue.FlexibleString.UnmarshalJSON` implements `json.Unmarshaler` and is invoked by `encoding/json` **via reflection**, which RTA cannot observe. It is used by three drivers' config structs; deleting it would break config decoding. → **Reflection/registry paths must be verified by hand.**

### (b) `deadcode` only reports functions — types and constants need manual checks

A file can be **entirely** dead while the tool shows only two functions:

```bash
grep -rn "PauseReason" --include=*.go . | grep -v pause_reason.go   # → empty
cat internal/domain/pause_reason.go                                # → 17 lines: type + 3 consts + 2 funcs, ALL dead
```

---

## Rule 3: Text reference-counting must strip comments

For frontend files there is no tool, so basename/stem counting is the fallback. **Comments are not references** — counting them produces **false negatives**:

- Measured: `web/src/composables/useTimeWindowSchedule.ts` appeared "referenced" only because a *dead* file mentioned it in **two comments** (lines 5 and 15). It had been dead all along; deleting the component revealed it.

```python
s = re.sub(r"/\*.*?\*/", "", s, flags=re.S)   # block comments
s = re.sub(r"<!--.*?-->", "", s, flags=re.S)  # Vue template comments
s = re.sub(r"(?m)//.*$", "", s)               # line + trailing comments
```

Also exclude entry points and declaration files (`main.ts`, `App.vue`, `*.d.ts`) from both the scan and the reference corpus.

---

## Rule 4: Iterate to a fixed point

Deleting dead code can orphan its only users. After **each** deletion round, re-scan; stop only when a round reports **no new** items.

Measured: deleting a component revealed a composable that had been masked by a comment (Rule 3).

---

## Rule 5: Prove frontend deletions with build artefacts

`grep` cannot prove a file is absent from the bundle (dynamic/indirect references are invisible to it). Rebuild and compare:

```bash
cd web && npm run build
cd .. && git status --short internal/api/web    # must be EMPTY
```

- **Empty** ⇒ the deleted file was never in the bundle ⇒ deletion is safe
- **Non-empty** ⇒ it **was** in the bundle ⇒ **revert that deletion** and re-judge

This is the only objective, self-proving check for frontend deletions — use it every time.

---

## Rule 6: `unused` stays out of the quality gate

Run it **ad-hoc**; do **not** add it to `.golangci.yml`. Like the browser-acceptance rule in `web/frontend/quality-guidelines.md`, it is a **diagnostic tool, not a gate**:

- it has systematic false-positive classes (interface stubs, test doubles) that require human adjudication
- gating on it would push contributors to litter `//nolint` instead of thinking

---

## Commands (reproducible)

```bash
export PATH=/usr/local/go/bin:/root/go/bin:$PATH
go install golang.org/x/tools/cmd/deadcode@latest

deadcode ./cmd/litepan                                    # production view
deadcode -test ./...                                      # + test roots (diff vs prod = test-only)
golangci-lint run --enable=unused -c .golangci.yml ./...   # unused identifiers (per package, incl. tests)
```

---

## Pre-Deletion Checklist

- [ ] **Both** tools run, results **compared** (not merged blindly)
- [ ] Every interface-looking method: **instantiation checked** (`grep` for the type being constructed/assigned)
- [ ] Types/constants of any "dead function's" file checked **by hand**
- [ ] Frontend reference counting **strips comments** before counting
- [ ] Re-scan after each deletion round; continue until a round is clean
- [ ] Frontend deletions proven by **build-artefact zero-churn**
- [ ] Nothing deleted that a **reflection / `init()` registry / string-keyed** path uses
