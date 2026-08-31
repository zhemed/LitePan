# 报告：任务设置单项填充

**任务**：`08-31-investigate-task-settings-padding`  
**截图**：`b33a32 1583x719`（单项 `任务并发` 左置，右侧空白不协调） vs `2ffaf0 895x414`（旧 4 项 `并发/限速/磁力/临时目录` 充实，仅作密度参考）  
**基线**：`main 7439a18→07c2d51 v0.0.7` `UploadTaskSettingsPanel.vue 309 行` `grid:1fr 1fr gap14px width720px`  
**约束**：**并发控件 `− 5 +` 格式不改**（用户已标注原有格式）

---

## 一、现状

| 项 | 现状 |
|---|---|
| **模板** | `<div class="task-settings__grid"><div class="task-settings__item--stepper">任务并发 − 5 +</div></div>` 仅 1 项 |
| **CSS** | `.task-settings__grid { grid:1fr 1fr; gap14px }` `width720px` `padding18px` |
| **效果** | 单卡片占左半 `~350px`，右半空白 `~350px`，`1583x719` 红框内大面积灰底留白，不协调（旧 4 项时 `2列` 填满） |
| **并发格式** | `stepper 38px 44px 38px` `− 5 +` 已定，不动 |

**来源**：`web/src/components/upload/UploadTaskSettingsPanel.vue:91-117` `193-197`

---

## 二、填充方案（3 选，不改并发）

| 方案 | 改动 | 效果 | 成本/风险 |
|---|---|---|---|
| **A 单项全宽居中（推荐）** | `.task-settings__item--stepper { grid-column:1/-1; max-width:420px; margin:0 auto; }` 或 `grid:1fr` 单列；`task-settings__grid` 在单项时 `grid-template-columns:1fr` | 单卡片居中占满 `~684px`（`720-36`），视觉饱满，无空白；并发 `− 5 +` 原样居右 | **1 处 CSS**，`vue-tsc 0`，`768px` 响应式已 `1fr` 无冲突 |
| **B 新增说明占位卡（次选）** | 右侧 `1fr` 加 ` <div class="task-settings__item is-help">` 说明 `三个队列独立使用此上限，1–5` + 图标，保持 `2列` | 2 卡并列，密度近旧 `2ffaf0`（左并发/右说明），信息增益 | **+30 行模板+CSS**，需文案 |
| **C 增高留白+居中** | `task-settings__grid { grid:1fr; place-items:center; min-height:120px }` `item { width:420px }` | 单项垂直居中，留白上下而非右侧，更平衡 | **1 处 CSS**，与 A 类似但垂直 |

**共同**：`width720px` 可保持或微缩至 `520px`（单项时 `max-width` 控制），`354-2d` 过渡不变。

---

## 三、推荐

**A** 最小改动、零文案、并发格式 100% 保留，已标注原有格式无需动。

**实施**（约 5 行）：
```css
.task-settings__grid:has(> :only-child) {
  grid-template-columns: 1fr;
  place-items: center;
}
.task-settings__item--stepper:only-child {
  width: min(420px, 100%);
  grid-column: 1 / -1;
  margin: 0 auto;
}
```
或更兼容（无 `:has`）：
```css
.task-settings__grid {
  grid-template-columns: 1fr;
}
.task-settings__item--stepper {
  max-width: 420px;
  margin: 0 auto;
  grid-column: 1 / -1;
}
```

**验证**：`1583x719` 单项居中后左右留白 `~150px` 对称，`895x414` 旧密度参考不再空；`768px` 下 `1fr` 已单列，无额外断点。

---

## 四、是否可设置填充

**是**。仅 `web` CSS/模板填充，无后端，无 `go vet` 影响，`type-check` 0。

---

## 附：取证

```bash
sed -n '90,120p' web/src/components/upload/UploadTaskSettingsPanel.vue
grep -n "grid-template-columns" web/src/components/upload/UploadTaskSettingsPanel.vue
```
