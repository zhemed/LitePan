/**
 * 0.0.29：任务计数分桶。
 *
 * 背景：0.0.27 引入服务端计数后，徽标/导航用 `total - success - skipped` 当"进行中"，
 * 把 paused/failed/canceled 全算成"上传中"——有暂停任务时会显示"上传中 1810"（误导）。
 *
 * 语义（与面板列表内容保持一致）：
 *   running = pending + running        → 徽标"上传中 N"
 *   paused  = paused                   → 徽标"已暂停 N"
 *   failed  = failed + canceled        → 徽标"失败 N"
 *   done    = success + skipped        → 徽标"上传完成 N"
 *   active  = running + paused         → 导航"进行中"（列表里暂停项也归此组）
 */
export interface UploadTaskTotals {
  running: number;
  paused: number;
  failed: number;
  done: number;
  active: number;
}

export function computeUploadTaskTotals(counts: Record<string, number> | null | undefined): UploadTaskTotals {
  const c = counts || {};
  const num = (key: string) => Number(c[key] || 0);
  const running = num("pending") + num("running");
  const paused = num("paused");
  const failed = num("failed") + num("canceled");
  const done = num("success") + num("skipped");
  return { running, paused, failed, done, active: running + paused };
}

/** 徽标文案：按 运行中 → 已暂停 → 失败 → 已完成 的优先级展示。 */
export function uploadTaskBadgeText(totals: UploadTaskTotals): string {
  if (totals.running > 0) return `上传中 ${totals.running}`;
  if (totals.paused > 0) return `已暂停 ${totals.paused}`;
  if (totals.failed > 0) return `失败 ${totals.failed}`;
  if (totals.done > 0) return `上传完成 ${totals.done}`;
  return "";
}
