/**
 * 上传任务「暂停交付 / 展示状态」纯函数（0.0.33）。
 *
 * 背景：生产实测「账号冷却等待中的任务点暂停不生效」——前端在两种情况下会
 * 静默跳过服务端暂停请求（任务在待恢复集合里 / 本地乐观状态过期），而服务端
 * 其实具备立即暂停能力（暂停会 cancel 冷却等待）。这里把判定收敛为无依赖纯函数，
 * 既让两条入口（行内主按钮、批量选择）共用同一规则，也便于断言脚本覆盖。
 */

/** 终态：服务端不会再执行，无需发送暂停。 */
export const TERMINAL_UPLOAD_STATUSES = ["success", "skipped"] as const;

export function isTerminalUploadStatus(status: unknown): boolean {
  return (TERMINAL_UPLOAD_STATUSES as readonly string[]).includes(String(status ?? ""));
}

/** 服务端冷却文案（契约：与 internal/file/service.go 的「账号网络冷却」措辞耦合）。 */
export function isCooldownMessage(message: unknown): boolean {
  return /冷却/.test(String(message ?? ""));
}

export type PauseCollectable = { task_id?: unknown; status?: unknown };

/**
 * 收集需要发送服务端暂停的远程任务 id。
 *
 * 规则：跳过本地（浏览器内）任务、空 id、终态；按 id 去重。
 * **不参考** pending/paused/running 的本地判断——本地乐观状态可能过期，
 * 陈旧 paused 也必须入选（服务端对已暂停任务是幂等 no-op）。
 */
export function collectRemotePauseIds<T extends PauseCollectable>(
  tasks: readonly T[],
  isLocalTask: (task: T) => boolean,
): string[] {
  const ids: string[] = [];
  const seen = new Set<string>();
  for (const task of tasks) {
    if (isLocalTask(task)) continue;
    const id = String(task.task_id ?? "");
    if (!id || seen.has(id)) continue;
    if (isTerminalUploadStatus(task.status)) continue;
    seen.add(id);
    ids.push(id);
  }
  return ids;
}

export interface UploadDisplayStatusContext {
  /** 任务 id 是否仍在客户端「待恢复」集合中。 */
  waitingResume: boolean;
  /** 任务最后一条消息是否表达冷却等待（服务端真相优先）。 */
  cooling: boolean;
}

/**
 * 解析任务展示状态。
 *
 * - 冷却等待：以**服务端状态**为准（不再被待恢复集合掩码成 pending，避免
 *   「明明在冷却重试却显示等待继续」，用户据此点暂停却看不出效果）；
 * - 其余：维持既有语义——待恢复集合内的 paused/failed/canceled 展示为 pending
 *  （客户端马上会把它 resume，提前反馈）。
 */
export function resolveUploadDisplayStatus(status: string, ctx: UploadDisplayStatusContext): string {
  if (ctx.cooling) return status;
  if (ctx.waitingResume && ["paused", "failed", "canceled"].includes(status)) return "pending";
  return status;
}
