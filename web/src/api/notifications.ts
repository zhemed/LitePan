import { http } from "./client";

export const NOTIFICATION_CATEGORY_CACHE_SCOPE_WARN = "cache_scope_warn";

export interface NotificationItem {
  id: number;
  level: string;
  category: string;
  title: string;
  message: string;
  account_id?: number;
  ref_id?: number;
  is_read: boolean;
  created_at: string;
}

export async function fetchNotifications(params?: { limit?: number; offset?: number }) {
  return http.get<{ items: NotificationItem[] }>("/admin/notifications", params);
}

export async function fetchUnreadCount() {
  return http.get<{ count: number }>("/admin/notifications/unread-count");
}

export async function markNotificationRead(id: number) {
  await http.post<Record<string, never>>(`/admin/notifications/${id}/read`);
}

export async function markAllNotificationsRead() {
  return http.post<{ marked: number }>("/admin/notifications/read-all");
}

export async function deleteNotification(id: number) {
  await http.del<Record<string, never>>(`/admin/notifications/${id}`);
}

export async function deleteAllNotifications() {
  return http.del<{ deleted: number }>("/admin/notifications");
}

export function isCacheScopeWarnNotification(item: NotificationItem): boolean {
  if (item.category === NOTIFICATION_CATEGORY_CACHE_SCOPE_WARN && (item.ref_id ?? 0) > 0) {
    return true;
  }
  return item.category === "cache" && item.title.includes("范围过大") && (item.ref_id ?? 0) > 0;
}
