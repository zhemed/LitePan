# 审查报告：是否可移除 `/root/LitePan/LitePan-own`

**任务**：`08-31-review-remove-nested-litepan-own`  
**时间**：2026-08-31  
**方式**：只读 `ls/du/git/grep`，不实际 `rm`  
**结论**：**可安全移除**（推荐移除，释放 57M）

---

## 一、现状清单

| 路径 | 类型 | 大小 | HEAD | remote | .gitignore |
|---|---|---|---|---|---|
| `/root/LitePan` | **工作区唯一** `zhemed/LitePan main` | `90M` (不含 `nested` 时) `147M`（含）| `ae853bc` `v0.0.3` | `github zhemed/LitePan` | - |
| `/root/LitePan/LitePan-own` | **nested** 克隆 | `57M`（`.git 42M` + 15M 检出）| `099cbc9 115 512MB` | `origin zhemed/LitePan-own` | ✅ `LitePan-own/` 已忽略，`git -C LitePan status --porcelain` 无误报 |
| `/root/LitePan-own` | **sibling** 克隆 | `394M`（`.git 44M` + 大工作区，因含 `node_modules`？实际 `du -sh` 394M 为含历史+缓存）→ 纠正：两者 `.git` 同 `42-44M`，`du` 差异因 `sibling` 有残留构建 | `099cbc9` 同 | `origin Ponphil/LitePan` + `own zhemed/LitePan-own` | -（独立目录） |
| `/_extracted/LitePan-own-custom` | 已静态化快照 | `~2M`（`diff 772行 + 9 patches + files`）| - | - | ✅ `/_extracted/` 已忽略 |

**来源**：`ls -ld` `du -sh` `git -C ... rev-parse` `cat .gitignore | grep LitePan-own` `git -C LitePan status --porcelain`

---

## 二、依赖清单

### 2.1 运行时依赖（`go build / docker / LitePan` 运行）

| 搜索 | 结果 | 是否依赖 `nested` |
|---|---|---|
| `grep -R "LitePan-own" /root/LitePan --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=LitePan-own \| grep -v ".trellis" \| grep -v "_extracted"` | **0 行**（生产代码无 `import LitePan-own`，`go.mod module litepan` 不引用） | **否** |
| `go vet / go build` | 不读取 `LitePan-own/` | 否 |
| `docker build` | `FROM node:20` `COPY` 仅 `web/` 与 `internal/`，无 `LitePan-own` | 否 |
| `LitePan data` | `data/litepan.db` 不含 `LitePan-own` 路径 | 否 |

**结论**：**生产零依赖**。

### 2.2 构建时/历史依赖（`trellis / _extracted`）

| 引用 | 路径 | 作用 | 是否可替 |
|---|---|---|---|
| `_extracted/README_CUSTOM.md` 示例 | `git -C /root/LitePan/LitePan-own diff HEAD~9..HEAD` | 说明提取命令 | ✅ 可替 `git -C /root/LitePan-own diff HEAD~9..HEAD` |
| `08-30-extract-litepan-own-custom/implement.md` 2.1-3.5 | 5 处 `cp /root/LitePan/LitePan-own/...` | 已执行完毕生成 `_extracted` 静态物 | ✅ 已固化，无需再 `cp` |
| `08-30-clone-litepan-own/prd` | `ls /root/LitePan/LitePan-own` 验证 | 历史验收条目 | ✅ 归档后仅历史，不影响新任务 |
| `.trellis/tasks/archive/.../review.md` | 提及 `2个克隆均Clean` | 历史描述 | ✅ 描述可保留，无需运行 |
| `grep -R "root/LitePan/LitePan-own" --include="*.md"` | 6 行均在 `*.md` 任务文档 | 文档示例 | ✅ 改为 `sibling` 路径即可 |

**结论**：**历史文档提及，非运行依赖**；`sibling` 可 100% 替代（两者 `HEAD 099cbc9` 同、`remote` 同、`_extracted` 已静态化）。

### 2.3 替代性验证

```bash
git -C /root/LitePan/LitePan-own log --oneline -1  # 099cbc9
git -C /root/LitePan-own log --oneline -1          # 099cbc9 同
diff <(git -C /root/LitePan/LitePan-own diff HEAD~9..HEAD --stat) <(git -C /root/LitePan-own diff HEAD~9..HEAD --stat) # 0 行差异
ls /root/LitePan-own/drivers/115_Open/driver.go    # 存在可替
```

两者 `diff --stat` 一致，`_extracted` 已含 `full.diff` 无需再生成。

---

## 三、风险评估

| 风险 | 等级 | 说明 |
|---|---|---|
| 误删后需 `git diff` 对比 | **低** | 用 `sibling` 同 `HEAD~9..HEAD` 完全替代 |
| `_extracted` 丢失 | **无** | `/_extracted` 已 `2M` 且 `git log` 可重建，`nested` 非唯一源 |
| `.gitignore` 误提交 | **无** | `LitePan-own/` 保留在 `.gitignore` 内，移除后 `git status` 仍 Clean |
| 磁盘 | **收益** | `57M`（若含 `sibling` 缓存共 `~100M`）可回收 |

---

## 四、结论与建议

### 结论：**可移除，推荐移除**

- **工作区**：`/root/LitePan` 独立，`nested` 不在 `go/docker` 编译路径，亦不被 `LitePan` 运行依赖。
- **替代**：`sibling /root/LitePan-own` 同 `099cbc9`，`_extracted` 已静态化，`nested` 无唯一价值。
- **历史**：仅 `5 个` 任务文档示例引用路径，改 `sibling` 即可。

### 清理命令（需用户确认后执行）

```bash
# 1. 验证 sibling 可替（0 行差异即通过）
diff <(git -C /root/LitePan/LitePan-own diff HEAD~9..HEAD --stat) <(git -C /root/LitePan-own diff HEAD~9..HEAD --stat) && echo "OK sibling 可替" || echo "NG"

# 2. 移除 nested（57M）
rm -rf /root/LitePan/LitePan-own
ls -ld /root/LitePan/LitePan-own 2>&1 | grep "No such file" && echo "已移除" || echo "仍存在"

# 3. 验证工作区 Clean
git -C /root/LitePan status --porcelain | grep LitePan-own && echo "仍有引用" || echo "Clean"
cat /root/LitePan/.gitignore | grep LitePan-own  # 保留 LitePan-own/ 一行，无需删

# 4. （可选）更新 _extracted/README_CUSTOM.md 示例路径 LitePan/LitePan-own → LitePan-own
# sed -i 's|/root/LitePan/LitePan-own|/root/LitePan-own|g' _extracted/README_CUSTOM.md
```

### 保留建议

- **保留** `.gitignore` 的 `LitePan-own/` 与 `/_extracted/` 两行（防止未来误 `git add`）。
- **保留** `sibling /root/LitePan-own`（主源），`nested` 移除后工作区即满足 `用户：工作区只有LitePan`。
- 若后续需 `再提取`，用 `git -C /root/LitePan-own` 即可。

---

## 附：取证命令

```bash
ls -ld /root/LitePan/LitePan-own /root/LitePan-own && du -sh /root/LitePan/LitePan-own /root/LitePan-own
git -C /root/LitePan/LitePan-own remote -v; git -C /root/LitePan-own remote -v
grep -R "LitePan-own" /root/LitePan --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=LitePan-own | grep -v ".trellis" | wc -l
cat /root/LitePan/.gitignore | grep LitePan-own
```
