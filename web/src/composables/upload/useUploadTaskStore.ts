import { computed, reactive, ref } from "vue";
import type { UploadTask } from "@/types/upload";
import { getUploadTaskStableKey } from "@/composables/upload/uploadTaskFormatters";
import type { LocalUploadPayload, UploadTaskDeps } from "@/composables/upload/uploadTaskTypes";

const LOCAL_UPLOAD_SESSION_KEY = "litepan.local-upload-tasks";

type PersistedLocalUploadTask = Pick<
  UploadTask,
  "task_id" | "account_id" | "account_name" | "file_name" | "target_path" | "target_display_path" | "status" | "progress" | "uploaded_bytes" | "total_bytes" | "message" | "error"
>;

export function useUploadTaskStore(deps: UploadTaskDeps) {
  const uploadTasks = ref<UploadTask[]>([]);
  const localUploadTasks = ref<UploadTask[]>([]);
  const uploadTaskPanelOpen = ref(false);
  const taskPanelCategory = ref<"upload" | "relay">("upload");
  const uploadTaskPanelLoading = ref(false);
  const uploadTaskPanelLoadingText = ref("正在准备上传任务...");
  const uploadTaskOrderMap = ref<Record<string, number>>({});
  const uploadTaskServerConcurrency = ref(3);
  const batchPauseInProgress = ref(false);
  const pendingDirRefreshBatches = ref<Record<string, { count: number; creationRefreshed: boolean }>>({});

  function registerDirRefreshBatch(key: string, count: number) {
    pendingDirRefreshBatches.value = {
      ...pendingDirRefreshBatches.value,
      [key]: { count, creationRefreshed: false },
    };
  }

  function markDirRefreshBatchCreated(key: string) {
    const cur = pendingDirRefreshBatches.value[key];
    if (!cur || cur.creationRefreshed) return;
    pendingDirRefreshBatches.value = {
      ...pendingDirRefreshBatches.value,
      [key]: { ...cur, creationRefreshed: true },
    };
  }

  function resolveDirRefreshBatch(key: string) {
    if (!(key in pendingDirRefreshBatches.value)) return;
    const next = { ...pendingDirRefreshBatches.value };
    delete next[key];
    pendingDirRefreshBatches.value = next;
  }

  let uploadTaskOrderCounter = 0;

  const localUploadTaskControllers = new Map<string, AbortController>();
  const localUploadTaskPayloads = new Map<string, LocalUploadPayload>();
  const canceledLocalUploadTaskIds = new Set<string>();
  const pausedLocalUploadTaskIds = new Set<string>();
  const localDispatchingTaskIds = new Set<string>();
  const pendingRemoteResumeTaskIds = new Set<string>();
  const hiddenUploadTaskKeys = reactive(new Set<string>());
  let folderUploadRefreshPending = false;

  function uploadAffectsCurrentDirectory(task: UploadTask, currentPath: string) {
    if (String(task.account_id) !== String(deps.selectedAccountId.value)) return false;
    if (task.status !== "success" && task.status !== "skipped") return false;
    const parentId = String(task.result?.parent_id ?? task.target_path ?? "");
    return parentId === currentPath || task.target_path === currentPath;
  }

  function ensureUploadTaskDisplayOrder(task: UploadTask) {
    const key = getUploadTaskStableKey(task);
    if (!key || uploadTaskOrderMap.value[key]) return;
    const preferred = Number(task.queue_order || 0);
    const next = preferred > 0 ? preferred : uploadTaskOrderCounter + 1;
    uploadTaskOrderCounter = Math.max(uploadTaskOrderCounter, next);
    uploadTaskOrderMap.value = { ...uploadTaskOrderMap.value, [key]: next };
  }

  const displayUploadTasks = computed(() => {
    const merged = [...localUploadTasks.value, ...uploadTasks.value].filter(
      (t) => !hiddenUploadTaskKeys.has(getUploadTaskStableKey(t)),
    );
    return [...merged].sort((a, b) => {
      const oa = uploadTaskOrderMap.value[getUploadTaskStableKey(a)] ?? Number.MAX_SAFE_INTEGER;
      const ob = uploadTaskOrderMap.value[getUploadTaskStableKey(b)] ?? Number.MAX_SAFE_INTEGER;
      return oa - ob;
    });
  });

  const relayTasks = computed(() =>
    displayUploadTasks.value.filter(
      (task) => task.source_type === "cross_transfer" && task.phase === "downloading",
    ),
  );

  const activeRelayTasks = computed(() =>
    relayTasks.value.filter((task) => ["pending", "running", "paused"].includes(task.status)),
  );

  const failedRelayTasks = computed(() =>
    relayTasks.value.filter((task) => ["failed", "canceled"].includes(task.status)),
  );

  const activeRelayCount = computed(() => activeRelayTasks.value.length);

  const activeUploadTasks = computed(() =>
    displayUploadTasks.value.filter(
      (task) =>
        !(task.source_type === "cross_transfer" && task.phase === "downloading") &&
        (task.status === "pending" || task.status === "running"),
    ),
  );
  const uploadTaskBadgeText = computed(() => {
    const running = activeUploadTasks.value.length;
    if (running > 0) return `上传中 ${running}`;
    if (activeRelayCount.value > 0) return `跨盘中 ${activeRelayCount.value}`;
    const failed = displayUploadTasks.value.filter((t) => t.status === "failed").length;
    if (failed > 0) return `失败 ${failed}`;
    const paused = displayUploadTasks.value.filter((t) => t.status === "paused").length;
    if (paused > 0) return `已暂停 ${paused}`;
    const success = displayUploadTasks.value.filter((t) => t.status === "success").length;
    if (success > 0) return `上传完成 ${success}`;
    return "";
  });

  const uploadTaskLabel = computed(() => uploadTaskBadgeText.value || "暂无传输任务");
  function createLocalUploadTask(file: File, options: Partial<UploadTask> = {}): UploadTask {
    return {
      task_id: `local-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      account_id: deps.selectedAccountId.value as number,
      account_name: deps.selectedAccountName.value,
      file_name: options.file_name || file.name,
      target_path: options.target_path || deps.currentPath.value,
      target_display_path: options.target_display_path || "",
      status: "pending",
      progress: 0,
      uploaded_bytes: 0,
      total_bytes: file.size,
      message: "等待发送到 LitePan 服务器",
      error: "",
    };
  }

  function createSkippedUploadTask(file: File, reason: string, options: Partial<UploadTask> = {}): UploadTask {
    return {
      ...createLocalUploadTask(file, options),
      task_id: `local-skip-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      status: "skipped",
      message: reason,
    };
  }

  function addLocalUploadTask(task: UploadTask) {
    ensureUploadTaskDisplayOrder(task);
    localUploadTasks.value = [task, ...localUploadTasks.value];
    persistLocalUploadTasks();
  }

  function updateLocalUploadTask(taskId: string, patch: Partial<UploadTask>) {
    localUploadTasks.value = localUploadTasks.value.map((t) =>
      t.task_id === taskId ? { ...t, ...patch } : t,
    );
    persistLocalUploadTasks();
  }

  function removeLocalUploadTask(taskId: string) {
    localUploadTasks.value = localUploadTasks.value.filter((t) => t.task_id !== taskId);
    persistLocalUploadTasks();
  }

  function pruneLocalUploadTasksByStableKeys(keys: string[]) {
    if (!keys.length) return;
    const keySet = new Set(keys.filter(Boolean));
    if (!keySet.size) return;
    const next = localUploadTasks.value.filter((task) => !keySet.has(getUploadTaskStableKey(task)));
    if (next.length === localUploadTasks.value.length) return;
    localUploadTasks.value = next;
    persistLocalUploadTasks();
  }

  function serializeLocalUploadTask(task: UploadTask): PersistedLocalUploadTask {
    return {
      task_id: task.task_id,
      account_id: task.account_id,
      account_name: task.account_name,
      file_name: task.file_name,
      target_path: task.target_path,
      target_display_path: task.target_display_path,
      status: task.status,
      progress: Number(task.progress || 0),
      uploaded_bytes: Number(task.uploaded_bytes || 0),
      total_bytes: Number(task.total_bytes || 0),
      message: String(task.message || ""),
      error: String(task.error || ""),
    };
  }

  function persistLocalUploadTasks() {
    if (typeof window === "undefined") return;
    if (!localUploadTasks.value.length) {
      window.sessionStorage.removeItem(LOCAL_UPLOAD_SESSION_KEY);
      return;
    }
    const payload = localUploadTasks.value.map(serializeLocalUploadTask);
    window.sessionStorage.setItem(LOCAL_UPLOAD_SESSION_KEY, JSON.stringify(payload));
  }

  function restoreLocalUploadTasks() {
    if (typeof window === "undefined") return;
    const raw = window.sessionStorage.getItem(LOCAL_UPLOAD_SESSION_KEY);
    if (!raw) return;
    try {
      const parsed = JSON.parse(raw) as PersistedLocalUploadTask[];
      const restored = parsed.map((task) => ({
        ...task,
        status: "failed",
        progress: 0,
        message: "页面已刷新，本地投递已中断，请重新选择文件",
        error: "页面刷新后无法继续本地投递，请重新选择文件",
      })) as UploadTask[];
      restored.forEach((task) => ensureUploadTaskDisplayOrder(task));
      localUploadTasks.value = restored;
      persistLocalUploadTasks();
    } catch {
      window.sessionStorage.removeItem(LOCAL_UPLOAD_SESSION_KEY);
    }
  }

  function patchRemoteUploadTask(taskId: string, patch: Partial<UploadTask>) {
    uploadTasks.value = uploadTasks.value.map((t) => (t.task_id === taskId ? { ...t, ...patch } : t));
  }

  function removeRemoteUploadTask(taskId: string) {
    uploadTasks.value = uploadTasks.value.filter((t) => t.task_id !== taskId);
  }

  function markFolderUploadRefreshPending() {
    folderUploadRefreshPending = true;
  }

  function consumeFolderUploadRefreshPending() {
    const pending = folderUploadRefreshPending;
    folderUploadRefreshPending = false;
    return pending;
  }

  return {
    uploadTasks,
    localUploadTasks,
    uploadTaskPanelOpen,
    taskPanelCategory,
    uploadTaskPanelLoading,
    uploadTaskPanelLoadingText,
    uploadTaskOrderMap,
    uploadTaskServerConcurrency,
    batchPauseInProgress,
    localUploadTaskControllers,
    localUploadTaskPayloads,
    canceledLocalUploadTaskIds,
    pausedLocalUploadTaskIds,
    localDispatchingTaskIds,
    pendingRemoteResumeTaskIds,
    hiddenUploadTaskKeys,
    relayTasks,
    pendingDirRefreshBatches,
    registerDirRefreshBatch,
    markDirRefreshBatchCreated,
    resolveDirRefreshBatch,
    activeRelayTasks,
    failedRelayTasks,
    activeRelayCount,
    displayUploadTasks,
    activeUploadTasks,
    uploadTaskLabel,
    uploadAffectsCurrentDirectory,
    ensureUploadTaskDisplayOrder,
    createLocalUploadTask,
    createSkippedUploadTask,
    addLocalUploadTask,
    updateLocalUploadTask,
    removeLocalUploadTask,
    pruneLocalUploadTasksByStableKeys,
    persistLocalUploadTasks,
    restoreLocalUploadTasks,
    patchRemoteUploadTask,
    removeRemoteUploadTask,
    markFolderUploadRefreshPending,
    consumeFolderUploadRefreshPending,
  };
}

export type UploadTaskStore = ReturnType<typeof useUploadTaskStore>;
