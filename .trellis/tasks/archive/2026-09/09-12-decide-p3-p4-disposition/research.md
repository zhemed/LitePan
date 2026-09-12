# P3 / P4 处置裁定（零代码改动）

## 结论

| 项 | 内容 | 裁定 | 依据 |
|---|---|---|---|
| **P3** | `drivers/template`（438 行）+ httpx OAuth 辅助链路（5 个生产不可达符号） | **保留，不修改** | 见 §1 |
| **P4** | `POST /accounts/{id}/refresh-auth`（`internal/api/auth_refresh.go`，30 行） | **保留，不修改** | 见 §2 |

---

## 1. P3：`drivers/template` 是"测试脚手架 + 文档指定的驱动骨架"，不是遗留

证据（只读）：

```bash
$ grep -rn "drivers/template" --include="*.go" .
internal/auth/oauth_integration_test.go:14:	_ "litepan/drivers/template"
```

- 该空导入位于 **统一认证守卫的集成测试** 中，测试文件内注释写明：「本方适配（0.0.17）：123_Open/Baidu_Open/OneDrive 已在精简版移除，用同走标准代理信封与统一分类的 template 骨架驱动验证统一守卫」。删掉它，这个覆盖"OAuth 代理信封 + 失败分类（成功/失效/限流/上游故障/空响应）× 8 次重复调用"的测试将失去被测驱动。
- spec 明确把它当作新驱动骨架：`driver-development.md:71`「**Copy template**: `cp -r drivers/template drivers/FooCloud`」，并在 `:3`、`:45` 与 `directory-structure.md:13` 中列为保留驱动。
- httpx 的 `DoJSON` / `ParseDataEnvelope` / `PostOAuthProxyJSON` / `OAuthProxyHTTPError` / `OAuthProxyResponseError` 由 template 与 httpx 自身测试使用；它们在生产依赖图外，**Go 链接器不会把未被引用的函数打进二进制**（0.0.40 已实测：删除死包后镜像内二进制逐字节相同），因此保留成本≈0。

**结论**：删除会削弱测试覆盖并移除文档指定的驱动脚手架，收益（源码行数）远小于代价。**保留**。

---

## 2. P4：`refresh-auth` 是"运维预留端点"，删它等于改 HTTP 契约

证据（只读）：

```bash
$ grep -rn "refresh-auth" --include="*.go" --include="*.ts" --include="*.vue" --include="*.md" --include="*.sh" .
internal/api/router.go:180:  r.Post("/accounts/{id}/refresh-auth", h.refreshAccountAuth)   # 全仓唯一引用
```

- 前端账号相关动作只有 `refresh-profile`（`web/src/api/accounts.ts:12`，用于刷新账号资料），没有调用 `refresh-auth`。
- 无文档/脚本引用；handler 仅 30 行，行为＝对该账号执行一次被动认证刷新并返回结果。
- 用途：**人工/脚本强制刷新某账号认证**（例如令牌异常时用 curl 触发一次），属运维手段；系统本身已有自动刷新（认证调度器 + 守卫），不依赖该端点。
- 处置权衡：保留成本 ≈ 0（不进二进制、不占运行时资源）；删除会**改变已公开的 HTTP 接口面**，收益仅是 30 行代码。

**结论**：**保留**，并在本记录中登记其"运维预留"属性；若将来确认长期无人使用，可在下一次 API 收敛任务中随其它端点一起删除（届时应作为有意的契约变更处理）。

---

## 3. 本轮动作

- 零代码改动（`git status` 仅本任务目录）。
- 两项裁定已写入本文件与 journal，作为排查报告的收尾（报告 §9 的 P3/P4 由此关闭）。
