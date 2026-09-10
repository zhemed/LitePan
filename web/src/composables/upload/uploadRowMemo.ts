import type { UploadTask } from "@/types/upload";
import type { UploadTaskTreeNode } from "@/composables/upload/uploadTaskTree";

/**
 * 0.0.28：任务行与批次节点的记忆化。
 *
 * 背景：SSE 增量（最快约 8 次/秒）会触发面板 computed 链重算，原先每次都把
 * 全部任务（窗口内可达数千条）重建为 PanelRow / 批次行，纯属重复劳动。
 *
 * 语义：行按 `task_id + updated_at` 命中；批次节点按结构签名命中
 * （条目数 + 最大 updated_at + 首个任务标识 + 状态分布），任一变化即重建。
 */

export interface MemoStats {
  builds: number;
  hits: number;
}

export interface RowMemo<T> {
  get(task: UploadTask): T;
  stats: MemoStats;
  clear(): void;
  size(): number;
}

interface RowEntry<T> {
  updatedAt: number;
  row: T;
}

/** 行级记忆化：未变化（updated_at 相同）的任务复用已构建行。 */
export function createRowMemo<T>(build: (task: UploadTask) => T, limit = 6000): RowMemo<T> {
  const cache = new Map<string, RowEntry<T>>();
  const stats: MemoStats = { builds: 0, hits: 0 };
  return {
    get(task: UploadTask): T {
      const key = String(task.task_id);
      const updatedAt = Number(task.updated_at || 0);
      const hit = cache.get(key);
      if (hit && hit.updatedAt === updatedAt) {
        stats.hits += 1;
        return hit.row;
      }
      const row = build(task);
      if (cache.size >= limit) {
        // 简单 FIFO 淘汰一批，避免引入 LRU 复杂度
        const drop = Math.max(1, Math.floor(limit / 4));
        let removed = 0;
        for (const k of cache.keys()) {
          cache.delete(k);
          if (++removed >= drop) break;
        }
      }
      cache.set(key, { updatedAt, row });
      stats.builds += 1;
      return row;
    },
    stats,
    clear() {
      cache.clear();
    },
    size() {
      return cache.size;
    },
  };
}

export interface NodeMemo<T> {
  get(node: UploadTaskTreeNode): T;
  stats: MemoStats;
  clear(): void;
  size(): number;
}

/** 批次/文件夹节点签名：条目数 + 最大 updated_at + 首任务标识 + 状态分布。 */
export function nodeSignature(node: UploadTaskTreeNode): string {
  const tasks = node.tasks || [];
  let maxUpdated = 0;
  let active = 0;
  let failed = 0;
  let done = 0;
  for (const task of tasks) {
    const updated = Number(task.updated_at || 0);
    if (updated > maxUpdated) maxUpdated = updated;
    switch (String(task.status)) {
      case "success":
      case "skipped":
        done += 1;
        break;
      case "failed":
      case "canceled":
        failed += 1;
        break;
      default:
        active += 1;
        break;
    }
  }
  const first = tasks[0];
  const firstKey = first ? `${first.task_id}:${Number(first.updated_at || 0)}` : "";
  return `${tasks.length}|${maxUpdated}|${firstKey}|${active}/${failed}/${done}|${node.name}`;
}

/** 批次节点记忆化：结构签名未变则复用已构建行。 */
export function createNodeMemo<T>(build: (node: UploadTaskTreeNode) => T, limit = 2000): NodeMemo<T> {
  const cache = new Map<string, { sig: string; row: T }>();
  const stats: MemoStats = { builds: 0, hits: 0 };
  return {
    get(node: UploadTaskTreeNode): T {
      const key = node.id;
      const sig = nodeSignature(node);
      const hit = cache.get(key);
      if (hit && hit.sig === sig) {
        stats.hits += 1;
        return hit.row;
      }
      const row = build(node);
      if (cache.size >= limit) {
        const drop = Math.max(1, Math.floor(limit / 4));
        let removed = 0;
        for (const k of cache.keys()) {
          cache.delete(k);
          if (++removed >= drop) break;
        }
      }
      cache.set(key, { sig, row });
      stats.builds += 1;
      return row;
    },
    stats,
    clear() {
      cache.clear();
    },
    size() {
      return cache.size;
    },
  };
}
