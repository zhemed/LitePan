<template>
  <div ref="panelRoot" class="upload-task-panel" :class="{ 'is-expanded': panelExpanded }">
    <div class="upload-task-panel-header">
      <div class="panel-title">任务面板</div>
      <div class="panel-head-actions">
        <AppIconButton
          icon="settings"
          label="设置"
          variant="ghost"
          size="sm"
          title="设置"
          class="head-icon head-icon-btn"
          @click.stop="settingsOpen = !settingsOpen"
        />
        <button class="head-icon" type="button" title="说明" aria-label="说明" @click="openUploadHelp">
          <svg viewBox="0 0 16 16" aria-hidden="true">
            <circle cx="8" cy="8" r="5.4"></circle>
            <path d="M8 6.2V8.35"></path>
            <circle cx="8" cy="10.9" r="0.55" fill="currentColor" stroke="none"></circle>
          </svg>
        </button>
        <span class="head-divider" aria-hidden="true"></span>
        <button class="head-icon" type="button" :title="panelExpanded ? '还原' : '最大化'" :aria-label="panelExpanded ? '还原' : '最大化'" @click="panelExpanded = !panelExpanded">
          <svg viewBox="0 0 16 16" aria-hidden="true">
            <rect x="2.25" y="2.25" width="11.5" height="11.5" rx="1.75"></rect>
          </svg>
        </button>
        <button class="head-icon" type="button" title="关闭" aria-label="关闭" @click="closeUploadTaskPanel">
          <svg viewBox="0 0 16 16" aria-hidden="true">
            <path d="M3.5 3.5L12.5 12.5"></path>
            <path d="M12.5 3.5L3.5 12.5"></path>
          </svg>
        </button>
        <div class="panel-settings-wrap">
          <UploadTaskSettingsPanel
            :open="settingsOpen"
            :server-concurrency="uploadTaskServerConcurrency"
            @update:server-concurrency="onConcurrencyUpdated"
            @close="settingsOpen = false"
          />
        </div>
      </div>
    </div>

    <div class="upload-task-panel-body">
      <aside class="panel-sidebar">
        <div class="sidebar-nav">
          <div v-for="category in navCategories" :key="category.key" class="nav-group">
            <button
              type="button"
              class="nav-item"
              :class="{ 'is-active': taskPanelCategory === category.key }"
              @click="taskPanelCategory = category.key"
            >
              <span class="nav-label">
                <span class="nav-glyph">
                  <SvgIcon :name="category.icon" :size="14" />
                </span>
                <span class="nav-text">{{ category.label }}</span>
              </span>
              <span v-if="category.count > 0" class="nav-count">{{ category.count }}</span>
            </button>

            <div v-if="taskPanelCategory === category.key" class="nav-sub">
              <button
                v-for="state in category.states"
                :key="state.key"
                type="button"
                class="nav-sub-item"
                :class="{ 'is-active': state.active }"
                @click="state.onClick"
              >
                <span class="nav-sub-label">
                  <span class="nav-sub-text">{{ state.label }}</span>
                </span>
                <span v-if="state.count > 0" class="nav-sub-count">{{ state.count }}</span>
              </button>
            </div>
          </div>
        </div>
      </aside>

      <section class="panel-content">
        <div class="content-top">
          <label class="search-box">
            <svg viewBox="0 0 16 16" aria-hidden="true">
              <circle cx="7" cy="7" r="4.5"></circle>
              <path d="M10.6 10.6L13 13"></path>
            </svg>
            <input v-model.trim="keyword" type="text" placeholder="搜索任务" />
          </label>

          <div class="top-actions">
            <button class="action-btn" type="button" :disabled="visibleRows.length === 0" @click="toggleSelectAll">
              {{ allVisibleSelected ? "清选" : "全选" }}
            </button>
            <button class="action-btn" type="button" :disabled="!canToggleSelected" @click="handleSelectedToggle">
              {{ selectedToggleLabel }}
            </button>
            <button class="action-btn warn" type="button" :disabled="!canDeleteSelected" @click="handleSelectedDelete">
              {{ deleteButtonLabel }}
            </button>
          </div>
        </div>

        <AppStateBlock
          v-if="showLoading"
          :message="loadingText"
          loading
          min-height="220px"
        />

        <template v-else>
          <div v-if="visibleRows.length > 0" class="table-head">
            <div>文件名</div>
            <div>来源</div>
            <div>状态</div>
            <div>{{ tailColumnLabel }}</div>
          </div>

          <div v-if="visibleRows.length > 0" class="task-list">
            <div
              v-for="row in visibleRows"
              :key="row.id"
              class="task-row"
              :class="{ 'is-selected': selectedTaskIds.has(row.id) }"
              @click="handleRowClick($event, row.id)"
            >
              <div class="task-row-main">
                <div class="file-cell">
                  <DriverIcon
                    class="driver-chip"
                    :logo="row.badgeLogo"
                    :color="row.badgeColor"
                    :name="row.badgeName"
                    :size="32"
                  />
                  <div class="file-name">{{ row.name }}</div>
                </div>
                <div class="source-cell">{{ row.source }}</div>
                <div class="status-cell" :class="row.statusClass">
                  <span v-if="showStatusPulse(row)" class="status-pulse" :class="statusPulseClass(row)" aria-hidden="true"></span>
                  <span class="status-text">{{ row.status }}</span>
                </div>
                <div class="tail-cell" :class="{ 'is-active': row.tailActive }">
                  <button
                    v-if="row.actionLabel"
                    type="button"
                    class="row-action-btn"
                    @click.stop="handleRowAction(row)"
                  >
                    {{ row.actionLabel }}
                  </button>
                  <span v-else>{{ row.tail }}</span>
                </div>
              </div>
              <div v-if="row.showProgress" class="row-progress-line">
                <UploadProgressInner
                  v-if="useSmoothUploadProgress(row)"
                  :task="row.raw as UploadTask"
                  class="task-row-progress-inner"
                />
                <span v-else :class="row.progressClass" :style="{ width: `${clampProgress(row.progress)}%` }"></span>
              </div>
            </div>
          </div>

          <AppStateBlock v-else :message="emptyText" min-height="220px" />

          <div v-if="currentRows.length > 0 && detailBarVisible" class="status-bar">
            <span>{{ detailPrimary }}</span>
            <span v-if="detailSecondary" class="accent">{{ detailSecondary }}</span>
            <span v-if="detailExtra">{{ detailExtra }}</span>
          </div>
        </template>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import AppStateBlock from "@/components/base/AppStateBlock.vue";
import DriverIcon from "@/components/driver/DriverIcon.vue";
import AppIconButton from "@/components/base/AppIconButton.vue";
import SvgIcon from "@/components/icons/SvgIcon.vue";
import UploadProgressInner from "@/components/upload/UploadProgressInner.vue";
import UploadTaskSettingsPanel from "@/components/upload/UploadTaskSettingsPanel.vue";
import { getUploadTaskStableKey } from "@/composables/upload/uploadTaskFormatters";
import { formatSize } from "@/utils/format";
import type { UploadTask } from "@/types/upload";
import type { useUploadTasks } from "@/composables/useUploadTasks";

type CategoryKey = "upload";
type StateKey = "active" | "done" | "failed";

type PanelRow = {
  id: string;
  kind: CategoryKey;
  raw: any;
  name: string;
  source: string;
  provider?: string;
  status: string;
  statusDetail: string;
  statusClass: string;
  tail: string;
  tailActive: boolean;
  progress: number;
  progressClass: string;
  showProgress: boolean;
  actionLabel?: string;
  sortOrder: number;
  badgeLogo: string;
  badgeName: string;
  badgeColor: string;
  searchText: string;
};

const props = defineProps<{
  uploadApi: ReturnType<typeof useUploadTasks>;
}>();

const api = props.uploadApi;

const {
  displayUploadTasks,
  uploadTaskPanelLoading,
  uploadTaskPanelLoadingText,
  getUploadTaskDriverBadge,
  getUploadTaskDisplayStatus,
  getUploadTaskPhaseLabel,
  getUploadTaskSpeedText,
  getUploadTaskStatusText,
  handleDeleteUploadTasks,
  handleUploadTaskPrimaryAction,
  openUploadNoticeFromPanel,
  closeUploadTaskPanel,
  refreshUploadTaskServerConcurrency,
  formatUploadPart,
} = api;

const taskPanelCategory = computed({
  get: () => api.taskPanelCategory.value as CategoryKey,
  set: (v: CategoryKey) => {
    api.taskPanelCategory.value = v;
  },
});

const uploadTaskServerConcurrency = computed({
  get: () => api.uploadTaskServerConcurrency.value as number,
  set: (v: number) => {
    api.uploadTaskServerConcurrency.value = v;
  },
});

const keyword = ref("");
const settingsOpen = ref(false);
const panelExpanded = ref(false);
const selectedTaskIds = ref<Set<string>>(new Set());
const uploadStateFilter = ref<StateKey>("active");
const panelRoot = ref<HTMLElement | null>(null);

function useFullscreenPanelLayout() {
  if (typeof window === "undefined") return false;
  return panelExpanded.value || window.innerWidth <= 768;
}

function updatePanelLeft() {
  if (typeof window === "undefined") return;
  if (useFullscreenPanelLayout()) {
    if (panelRoot.value) panelRoot.value.style.left = "";
    return;
  }
  if (panelRoot.value) {
    panelRoot.value.style.left = `${document.documentElement.clientWidth / 2}px`;
  }
}

function openUploadHelp() {
  settingsOpen.value = false;
  if (panelRoot.value && !useFullscreenPanelLayout()) {
    const rect = panelRoot.value.getBoundingClientRect();
    panelRoot.value.style.left = `${rect.left + rect.width / 2}px`;
  }
  openUploadNoticeFromPanel();
}

function onConcurrencyUpdated(value: number) {
  uploadTaskServerConcurrency.value = value;
}

function onDocumentClick(event: MouseEvent) {
  if (!settingsOpen.value) return;
  const target = event.target as HTMLElement | null;
  if (target?.closest(".panel-head-actions")) return;
  if (target?.closest(".overlay--nested")) return;
  settingsOpen.value = false;
}

const uploadTasks = computed<UploadTask[]>(() => (displayUploadTasks?.value || []) as UploadTask[]);

function uploadStateOf(task: UploadTask): StateKey {
  const status = uploadDisplayStatus(task);
  if (status === "success" || status === "skipped") return "done";
  if (status === "failed" || status === "canceled") return "failed";
  return "active";
}

function uploadStatusLabel(task: UploadTask) {
  const displayStatus = typeof getUploadTaskDisplayStatus === "function" ? getUploadTaskDisplayStatus(task) : task.status;
  const progress = preciseUploadProgress(task).toFixed(1);
  const message = String(task.message || "").trim();
  const stageLabel = uploadStageLabel(message);
  const phaseLabel = typeof getUploadTaskPhaseLabel === "function" ? getUploadTaskPhaseLabel(task) : "";
  if (displayStatus === "running") {
    if (stageLabel) return stageLabel;
    return `上传中 ${progress}%`;
  }
  if (displayStatus === "paused") return progress !== "0.0" ? `已暂停 ${progress}%` : "已暂停";
  if (displayStatus === "pending") {
    if (isLocalDispatchMessage(message)) return stageLabel || message;
    if (phaseLabel && phaseLabel !== "等待继续") return phaseLabel;
    return "等待上传";
  }
  if (displayStatus === "failed") return progress !== "0.0" ? `失败 ${progress}%` : "失败";
  return getUploadTaskStatusText(displayStatus);
}

function isLocalDispatchMessage(message: string) {
  if (!message) return false;
  if (message.includes("投递到 LitePan 服务器")) return true;
  if (message.includes("投递成功")) return true;
  if (message.includes("创建任务中")) return true;
  return false;
}

function isUploadStageMessage(message: string) {
  if (!message) return false;
  if (message.includes("SHA-256")) return true;
  if (message.includes("校验")) return true;
  if (message.includes("发起上传")) return true;
  if (message.includes("准备上传")) return true;
  if (message.includes("预上传")) return true;
  if (message.includes("获取上传凭证")) return true;
  if (message.includes("创建上传会话")) return true;
  if (message.includes("完成上传")) return true;
  if (message.includes("写入网盘")) return true;
  return false;
}

function uploadStageLabel(message: string) {
  if (!message) return "";
  if (message.includes("投递成功") || message.includes("创建任务中")) return "创建任务中";
  if (message.includes("投递到 LitePan 服务器")) return message;
  if (message.includes("SHA-256")) return "计算校验中";
  if (message.includes("校验")) return "校验中";
  if (message.includes("发起上传") || message.includes("准备上传") || message.includes("预上传")) return "准备上传中";
  if (message.includes("获取上传凭证") || message.includes("创建上传会话")) return "获取凭证中";
  if (message.includes("完成上传") || message.includes("写入网盘")) return "提交上传中";
  return "";
}

function uploadDisplayStatus(task: UploadTask) {
  return String(typeof getUploadTaskDisplayStatus === "function" ? getUploadTaskDisplayStatus(task) : task.status || "");
}

function uploadResumePending(task: UploadTask) {
  if (uploadDisplayStatus(task) !== "pending") return false;
  const phaseLabel = typeof getUploadTaskPhaseLabel === "function" ? getUploadTaskPhaseLabel(task) : "";
  if (phaseLabel === "等待继续") return true;
  return String(task.message || "").includes("准备继续");
}

function stripChunkDetail(message: string) {
  return String(message || "")
    .replace(/[，,、]\s*分片[（(]\s*\d+\s*\/\s*\d+\s*[)）]\s*$/u, "")
    .replace(/\s*分片[（(]\s*\d+\s*\/\s*\d+\s*[)）]\s*$/u, "")
    .trim();
}

function uploadStatusDetail(task: UploadTask) {
  const details: string[] = [];
  if (task.error) details.push(String(task.error));
  const message = stripChunkDetail(String(task.message || "").trim());
  if (message && !details.includes(message)) details.push(message);
  const part = details.length === 0 && typeof formatUploadPart === "function" ? formatUploadPart(task) : "";
  if (part && !details.includes(part)) details.push(part);
  return details.join(" · ");
}

function buildUploadRow(task: UploadTask): PanelRow {
  const badge = getUploadTaskDriverBadge(task);
  const completed = uploadStateOf(task) === "done";
  const displayStatus = uploadDisplayStatus(task);
  const retryable = displayStatus === "failed" || displayStatus === "canceled";
  const speed = getUploadTaskSpeedText(task) || "---";
  const progress = preciseUploadProgress(task);
  const message = String(task.message || "").trim();
  const localDispatching = displayStatus === "pending" && isLocalDispatchMessage(message);
  const stageRunning = displayStatus === "running" && isUploadStageMessage(message);
  return {
    id: getUploadTaskStableKey(task),
    kind: "upload",
    raw: task,
    name: task.file_name,
    source:
      task.source_type === "offline_handoff"
          ? "离线接棒"
          : task.source_type === "server_local"
            ? "服务器上传"
            : "手动上传",
    status: uploadStatusLabel(task),
    statusDetail: uploadStatusDetail(task),
    statusClass: displayStatus,
    tail: completed ? "" : speed,
    tailActive: !completed && speed !== "---",
    progress,
    progressClass: "uploading",
    showProgress: (displayStatus === "running" && progress > 0 && !stageRunning) || (displayStatus === "pending" && progress > 0 && !localDispatching),
    actionLabel: completed ? "打开" : retryable ? "重试" : undefined,
    sortOrder: taskOrder(task),
    badgeLogo: badge.logo || "",
    badgeName: String(badge.name || "网盘").slice(0, 2),
    badgeColor: badge.color || "#4b63e9",
    searchText: [task.file_name, task.account_name, task.target_display_path, task.message, task.error].join(" "),
  };
}

const uploadRows = computed(() => uploadTasks.value.map(buildUploadRow));

function stateFilterOf(_category: CategoryKey) {
  return uploadStateFilter.value;
}

const baseRows = computed(() => uploadRows.value);

const currentRows = computed(() =>
  [...baseRows.value]
    .filter((row) => uploadStateOf(row.raw) === stateFilterOf(taskPanelCategory.value))
    .sort((a, b) => a.sortOrder - b.sortOrder),
);

const visibleRows = computed(() => {
  const query = keyword.value.toLowerCase();
  if (!query) return currentRows.value;
  return currentRows.value.filter((row) => row.searchText.toLowerCase().includes(query));
});

watch(visibleRows, (rows) => {
  const visibleIds = new Set(rows.map((row) => row.id));
  selectedTaskIds.value = new Set([...selectedTaskIds.value].filter((id) => visibleIds.has(id)));
}, { immediate: true });

const selectedRows = computed(() => visibleRows.value.filter((row) => selectedTaskIds.value.has(row.id)));
const allVisibleSelected = computed(() => visibleRows.value.length > 0 && visibleRows.value.every((row) => selectedTaskIds.value.has(row.id)));
const detailRow = computed(() => (selectedRows.value.length === 1 ? selectedRows.value[0] : null));
const detailBarVisible = computed(() => Boolean(detailRow.value));

function resolveUploadTaskForRow(row: PanelRow): UploadTask | null {
  if (row.kind === "upload") return row.raw as UploadTask;
  return null;
}

const selectedToggleTasks = computed(() =>
  selectedRows.value
    .map(resolveUploadTaskForRow)
    .filter((task): task is UploadTask => {
      if (!task) return false;
      return ["pending", "running", "paused", "failed", "canceled"].includes(uploadDisplayStatus(task));
    }),
);

const canToggleSelected = computed(() => selectedToggleTasks.value.length > 0);

const selectedToggleLabel = computed(() => {
  if (!selectedToggleTasks.value.length) return "暂停";
  return selectedToggleTasks.value.every((task) => ["paused", "failed", "canceled"].includes(uploadDisplayStatus(task))) ? "继续" : "暂停";
});

const tailColumnLabel = computed(() => {
  if (taskPanelCategory.value === "upload" && uploadStateFilter.value === "done") return "操作";
  if (taskPanelCategory.value === "upload" && uploadStateFilter.value === "failed") return "操作";
  return "速度";
});

const emptyText = computed(() => {
  return uploadStateFilter.value === "done" ? "暂无已完成上传任务" : uploadStateFilter.value === "failed" ? "暂无失败上传任务" : "暂无进行中的上传任务";
});

const showLoading = computed(() => taskPanelCategory.value === "upload" && uploadTaskPanelLoading?.value);
const loadingText = computed(() => uploadTaskPanelLoadingText?.value || "正在加载上传任务...");

function countByState(_category: CategoryKey, state: StateKey) {
  const rows = uploadRows.value;
  return rows.filter((row) => uploadStateOf(row.raw) === state).length;
}

const navCategories = computed(() => [
  {
    key: "upload" as const,
    label: "上传列表",
    icon: "upload",
    count: countByState("upload", "active"),
    states: [
      { key: "active", label: "进行中", count: countByState("upload", "active"), active: uploadStateFilter.value === "active", onClick: () => { uploadStateFilter.value = "active"; } },
      { key: "done", label: "已完成", count: countByState("upload", "done"), active: uploadStateFilter.value === "done", onClick: () => { uploadStateFilter.value = "done"; } },
      { key: "failed", label: "失　败", count: countByState("upload", "failed"), active: uploadStateFilter.value === "failed", onClick: () => { uploadStateFilter.value = "failed"; } },
    ],
  }
]);

const detailPrimary = computed(() => {
  const row = detailRow.value;
  if (!row) return "";
  if (row.statusDetail) return row.statusDetail;
  return row.status;
});

const detailSecondary = computed(() => {
  const row = detailRow.value;
  if (!row) return "";
  return row.tailActive && row.tail !== "---" ? row.tail : "";
});

const detailExtra = computed(() => {
  const row = detailRow.value;
  if (!row) return "";
  const details: string[] = [];
  if (row.kind === "upload") {
    const task = row.raw as UploadTask;
    const part = typeof formatUploadPart === "function" ? formatUploadPart(task) : "";
    if (part) details.push(part);
    const total = Number(task.total_bytes || 0);
    const transferred = uploadTransferredBytes(task);
    if (total > 0 && transferred > 0) {
      details.push(`${formatSize(transferred)} / ${formatSize(total)}`);
    }
    return details.join(" · ");
  }
  return details.join(" · ");
});

function handleRowClick(event: MouseEvent, rowId: string) {
  if (event.metaKey || event.ctrlKey) {
    const next = new Set(selectedTaskIds.value);
    if (next.has(rowId)) next.delete(rowId);
    else next.add(rowId);
    selectedTaskIds.value = next;
    return;
  }
  selectedTaskIds.value = new Set([rowId]);
}

function showStatusPulse(row: PanelRow) {
  if (row.kind === "upload") {
    return row.statusClass === "pending" || row.statusClass === "running";
  }
  return false;
}

function statusPulseClass(row: PanelRow) {
  if (row.kind === "upload" && row.statusClass === "pending") {
    return "is-uploading";
  }
  return "";
}

function toggleSelectAll() {
  if (allVisibleSelected.value) {
    selectedTaskIds.value = new Set();
    return;
  }
  selectedTaskIds.value = new Set(visibleRows.value.map((row) => row.id));
}

const deletableSelectedRows = computed(() => selectedRows.value);
const canDeleteSelected = computed(() => deletableSelectedRows.value.length > 0);
const deleteButtonLabel = computed(() => "删除");

async function handleSelectedToggle() {
  const tasks = [...selectedToggleTasks.value];
  if (!tasks.length) return;
  const resumeMode = tasks.every((task) => ["paused", "failed", "canceled"].includes(uploadDisplayStatus(task)));
  for (const task of tasks) {
    const displayStatus = uploadDisplayStatus(task);
    if (resumeMode && ["paused", "failed", "canceled"].includes(displayStatus)) {
      await handleUploadTaskPrimaryAction(task);
    }
    if (!resumeMode && ["pending", "running"].includes(displayStatus)) {
      await handleUploadTaskPrimaryAction(task);
    }
  }
}

async function handleSelectedDelete() {
  const rows = [...selectedRows.value];
  if (!rows.length) return;
  await handleDeleteUploadTasks(rows.map((row) => row.raw as UploadTask));
  selectedTaskIds.value = new Set();
}

async function handleRowAction(row: PanelRow) {
  const task = resolveUploadTaskForRow(row);
  if (task) {
    await handleUploadTaskPrimaryAction(task);
    return;
  }
}

function clampProgress(value: number) {
  return Math.max(0, Math.min(100, Number(value || 0)));
}

function uploadTransferredBytes(task: UploadTask) {
  return Number(task.uploaded_bytes || 0);
}

function useSmoothUploadProgress(row: PanelRow) {
  if (row.kind !== "upload") return false;
  const task = row.raw as UploadTask;
  return Number(task.total_bytes || 0) > 0 && ["pending", "running", "paused"].includes(task.status);
}

function preciseUploadProgress(task: UploadTask) {
  const total = Number(task.total_bytes || 0);
  if (total > 0) {
    const transferred = uploadTransferredBytes(task);
    if (transferred > 0) {
      return clampProgress((transferred * 100) / total);
    }
  }
  return clampProgress(Number(task.progress || 0));
}

function taskOrder(task: UploadTask) {
  const displayStatus = uploadDisplayStatus(task);
  const rank = uploadStatusRank(task);
  const order = Number(task.queue_order || 0);
  const created = Number(task.created_at || 0);
  const updated = Number(task.updated_at || created || 0);
  if (uploadResumePending(task)) {
    return rank * 1_000_000_000_000 - updated;
  }
  if (displayStatus === "paused") {
    return rank * 1_000_000_000_000 + updated;
  }
  return rank * 1_000_000_000_000 + (order > 0 ? order : created);
}

function uploadStatusRank(task: UploadTask) {
  const status = uploadDisplayStatus(task);
  if (status === "running") return 0;
  if (uploadResumePending(task)) return 1;
  if (status === "paused") return 2;
  if (status === "pending") return 3;
  if (status === "failed") return 4;
  if (status === "canceled") return 5;
  if (status === "success") return 6;
  if (status === "skipped") return 7;
  return 9;
}

onMounted(() => {
  updatePanelLeft();
  window.addEventListener("resize", updatePanelLeft);
  document.addEventListener("click", onDocumentClick);
  void refreshUploadTaskServerConcurrency?.();
});

watch(panelExpanded, () => {
  updatePanelLeft();
});

onUnmounted(() => {
  window.removeEventListener("resize", updatePanelLeft);
  document.removeEventListener("click", onDocumentClick);
});
</script>

<style scoped src="@/styles/upload-task-panel.css"></style>
