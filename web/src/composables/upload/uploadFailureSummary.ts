import type { UploadTask } from "@/types/upload";

/**
 * 0.0.31：上传失败/重试的可读性。
 *
 * 背景：系统日志里大量「上传文件失败」其实是账号网络冷却（会自愈），前端任务行也
 * 只给状态不给"为什么"。这里把错误归类，供面板展示「失败 N · 网络异常 ×k …」，
 * 让用户一眼看出是否需要人工处理。
 */

export type UploadFailureKind =
  | "冷却重试"
  | "网络异常"
  | "认证失效"
  | "账号限流"
  | "权限不足"
  | "文件冲突"
  | "本地文件缺失"
  | "其他";

const KIND_RULES: Array<{ kind: UploadFailureKind; patterns: RegExp[] }> = [
  { kind: "冷却重试", patterns: [/account_cooldown/, /账号网络异常/, /冷却/] },
  { kind: "认证失效", patterns: [/AUTH_EXPIRED/, /认证已失效/, /会话已失效/, /token.*失效/i] },
  { kind: "账号限流", patterns: [/RATE_LIMITED/, /请求过于频繁/, /限流/] },
  { kind: "权限不足", patterns: [/PERMISSION_DENIED/, /权限不足/] },
  { kind: "网络异常", patterns: [/网络/, /timeout/i, /超时/, /connection/i, /HTTP 5\d\d/] },
  { kind: "文件冲突", patterns: [/已存在/, /冲突/, /InvalidPartOrder/i, /duplicate/i] },
  { kind: "本地文件缺失", patterns: [/本地文件/, /no such file/i, /文件不存在/] },
];

/** 把任务的错误/状态文案归类为可读原因；无法识别归"其他"。 */
export function classifyUploadFailure(error?: string, message?: string): UploadFailureKind {
  const text = `${error || ""} ${message || ""}`.trim();
  if (!text) return "其他";
  for (const rule of KIND_RULES) {
    if (rule.patterns.some((re) => re.test(text))) return rule.kind;
  }
  return "其他";
}

/** 判断任务是否处于"冷却重试中"（可自愈，非失败）。 */
export function isCooldownRetrying(task: UploadTask): boolean {
  const status = String(task.status || "");
  if (status !== "pending" && status !== "running") return false;
  const text = `${task.message || ""} ${task.error || ""}`;
  return /冷却/.test(text) || /account_cooldown/.test(text);
}

export interface UploadFailureSummary {
  total: number;
  parts: string[];
}

/** 统计失败/取消任务的原因分布，返回总量与占比最高的前 limit 类（形如「网络异常 10」）。 */
export function summarizeUploadFailures(tasks: UploadTask[], limit = 3): UploadFailureSummary {
  const counts = new Map<UploadFailureKind, number>();
  let total = 0;
  for (const task of tasks) {
    const status = String(task.status || "");
    if (status !== "failed" && status !== "canceled") continue;
    total += 1;
    const kind = classifyUploadFailure(task.error, task.message);
    counts.set(kind, (counts.get(kind) || 0) + 1);
  }
  const parts = [...counts.entries()]
    .sort((a, b) => (b[1] === a[1] ? a[0].localeCompare(b[0]) : b[1] - a[1]))
    .slice(0, Math.max(1, limit))
    .map(([kind, count]) => `${kind} ${count}`);
  return { total, parts };
}
