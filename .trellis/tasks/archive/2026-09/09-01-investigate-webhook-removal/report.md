# 报告：第三方通知Webhook是否可彻底移除

**任务**：`09-01-investigate-webhook-removal`  
**时间**：2026-09-01  
**基线**：`main 1a77f58 v0.0.10`  
**结论**：**可彻底移除**（`local_upload` 仅用 `daily/interval`，`webhook` 零耦合），**8 go + 约 15 前端** 需清理

---

## 一、清单（`grep webhook|第三方`）

| 层 | 文件 | 行 | 内容 |
|---|---|---|---|
| **domain** | `internal/domain/automation.go:19` | 1 | `AutomationTriggerWebhook="webhook"` |
| **service** | `internal/automation/service_webhook.go` 40行 | `TriggerWebhook` `matchWebhook` `SubmitRun webhook` | |
|  | `service_validate.go:75,94` | 2 | `TriggerWebhook` 白名单 + `event` 校验 |
|  | `service.go:105` `WebhookEvent` + `191,216,240` `NextRunAt` `webhook` 分支 | 4 | `WebhookEvent` 结构体 + 3 处 `TriggerWebhook` 判断 |
|  | `service_run.go:102` | 1 | `if TriggerWebhook` 清 `NextRunAt` |
| **api** | `internal/api/automation.go:190` `automationWebhook` + `decodeWebhookEvent` | 2 | `POST /automation/events` |
|  | `internal/api/router.go:147` | 1 | `r.Post("/automation/events", h.automationWebhook)` |
| **test** | `internal/automation/service_test.go:26` `TestTriggerWebhookQueuesEveryMatchedRule` + `webhookRule` | 2 | |
| **前端 API** | `web/src/api/automation.ts:3` `AutomationTriggerType "daily|interval|webhook"` | 1 | |
| **前端 视图** | `web/src/components/admin/AutomationPanel.vue:172,381,414,691,748,850,1194,1324,1448` | 9 | `第三方通知` 触发器卡、 `api-example`、 `trigger_type external_event ↔ webhook` 映射、 `event` 字段 |
|  | `web/src/components/admin/ApiKeySettings.vue:255,317` | 2 | `Webhook` 秘钥说明 |

**总数**：`8 go` + `~15 ts/vue`。

---

## 二、依赖图

```
外部程序 POST /api/automation/events (Bearer ApiKey)
  → api/automationWebhook → decodeWebhookEvent(event,source,path)
    → service.TriggerWebhook → matchWebhook(event) → submitRun(webhook)
      → 仅影响 webhook 触发的规则；daily/interval 不走此路径

AutomationPanel 第三方通知
  → trigger_type external_event ↔ webhook 映射
  → ApiKeySettings 秘钥供 webhook 鉴权
```

**与 `local_upload` 关系**：**零耦合**。`local_upload` 可配 `daily 00:05` / `interval 1h`，不需 `webhook`；`webhook` 仅为可选触发源。

---

## 三、能否彻底移除

| 判断 | 结论 | 依据 |
|---|---|---|
| **local_upload 是否依赖** | **否** | `runLocalUpload` 不读 `webhook` |
| **DB** | **可删** | `automation_rules trigger_type=webhook` 独立，无外键 |
| **ApiKey** | **可保留** | `ApiKey` 除 `webhook` 外无其他用途，`webhook` 删后 `ApiKey` 仍可但无触发，可一并保留或删 `webhook` 文案 |
| **前端** | **可删** | `第三方通知` 卡仅 UI |
| **测试** | **可删** | `TestTriggerWebhook` 专测 webhook |

**安全条件**：确认 **不再使用外部程序 Webhook 通知**（如 `CloudSaver` 等）。若仅 `daily/interval` 则可删。

---

## 四、彻底移除清单（若执行）

**后端 8**：
- `internal/domain/automation.go:19` `TriggerWebhook`
- `internal/automation/service_webhook.go` 整文件
- `internal/automation/service_validate.go:75` 白名单 `webhook` + `94` 分支
- `internal/automation/service.go:105` `WebhookEvent` + `191,216,240` 3 处
- `internal/automation/service_run.go:102`
- `internal/api/automation.go:190` `automationWebhook/decodeWebhookEvent`
- `internal/api/router.go:147`
- `internal/automation/service_test.go:26` `webhookRule` 及用例

**前端 4**：
- `web/src/api/automation.ts:3` `"webhook"` 从联合类型
- `web/src/components/admin/AutomationPanel.vue` `第三方通知` 9 处（`172,381,414,691,748,850,1194,1324,1448`）
- `web/src/components/admin/ApiKeySettings.vue:255,317` 文案 `Webhook` 可改通用

**DB**：`automation_rules WHERE trigger_type='webhook'`（可保留或 `UPDATE`）

---

## 五、建议

| 方案 | 操作 | 适用 |
|---|---|---|
| **A 彻底移除** | 按 §四 8+4 清单删，`go vet/type-check` 后 `0.0.11` | **推荐**：若仅 `daily/interval`，`local_upload` 不受影响 |
| **B 保留** | 不删 | 仍需 `Webhook` 联动 |
| **C 仅 UI** | 仅删 `AutomationPanel` 第三方卡 | 保留后端 `POST /automation/events` 供外部仍可调 |

**风险**：A 方案下历史 `webhook` 规则 `trigger_type` 需 `validate` 报错提示重配为 `daily/interval`。

---

## 附：取证

```bash
grep -R -n "Webhook" --include="*.go" --include="*.vue" --exclude-dir=_extracted | head
grep -R "AutomationTriggerWebhook" --include="*.go" | wc -l
```
