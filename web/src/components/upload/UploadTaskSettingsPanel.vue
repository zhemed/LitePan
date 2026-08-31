<script setup lang="ts">
import { onMounted, ref, watch } from "vue";
import { getApiErrorMessage } from "@/api/client";
import { uploadApi } from "@/api/upload";
import { toast } from "@/composables/useToast";

const props = defineProps<{
  open: boolean;
  serverConcurrency: number;
}>();

const emit = defineEmits<{
  "update:serverConcurrency": [number];
  close: [];
}>();

const loading = ref(true);
const loadedOnce = ref(false);
const transferSaving = ref(false);
const transferConcurrency = ref(3);
const transferMin = ref(1);
const transferMax = ref(5);

async function loadPanelSettings() {
  loading.value = !loadedOnce.value;
  const runtimeResult = await Promise.allSettled([uploadApi.getRuntime()]);
  if (runtimeResult[0].status === "fulfilled") {
    const data = (runtimeResult[0] as PromiseFulfilledResult<any>).value;
    transferConcurrency.value = data.concurrency;
    transferMin.value = data.concurrency_min ?? 1;
    transferMax.value = data.concurrency_max ?? 5;
    emit("update:serverConcurrency", data.concurrency);
  } else {
    transferConcurrency.value = props.serverConcurrency || 3;
  }
  loadedOnce.value = true;
  loading.value = false;
}

async function applyTransferConcurrency(next: number) {
  if (next < transferMin.value || next > transferMax.value || transferSaving.value) return;
  const previous = transferConcurrency.value;
  transferConcurrency.value = next;
  transferSaving.value = true;
  try {
    const data = await uploadApi.updateRuntime(next);
    transferConcurrency.value = data.concurrency;
    emit("update:serverConcurrency", data.concurrency);
    toast.success("任务并发已更新");
  } catch (error) {
    transferConcurrency.value = previous;
    toast.error(getApiErrorMessage(error, "更新任务并发失败"));
  } finally {
    transferSaving.value = false;
  }
}

function stepTransfer(delta: number) {
  void applyTransferConcurrency(transferConcurrency.value + delta);
}

watch(
  () => props.open,
  (open) => {
    if (open && !loading.value && !loadedOnce.value) {
      void loadPanelSettings();
    }
  },
);

onMounted(() => {
  void loadPanelSettings();
});
</script>

<template>
  <Transition name="task-settings">
    <div
      v-if="open && !loading"
      class="upload-settings-panel task-settings"
      role="dialog"
      aria-label="任务设置"
      @click.stop
    >
      <header class="task-settings__head">
        <strong>任务设置</strong>
        <button type="button" aria-label="关闭" @click="emit('close')">×</button>
      </header>

      <div class="task-settings__body">
        <div class="task-settings__grid">
          <div class="task-settings__item task-settings__item--stepper">
            <div class="task-settings__label">
              <strong>任务并发</strong>
              <small>三个队列独立使用此上限</small>
            </div>
            <div class="task-settings__stepper">
              <button
                type="button"
                :disabled="transferSaving || transferConcurrency <= transferMin"
                aria-label="减少任务并发"
                @click="stepTransfer(-1)"
              >
                −
              </button>
              <span>{{ transferConcurrency }}</span>
              <button
                type="button"
                :disabled="transferSaving || transferConcurrency >= transferMax"
                aria-label="增加任务并发"
                @click="stepTransfer(1)"
              >
                +
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.task-settings {
  position: absolute;
  top: calc(100% + 10px);
  right: 0;
  display: flex;
  flex-direction: column;
  width: 720px;
  max-width: calc(100vw - 32px);
  max-height: calc(min(720px, 86vh) - 58px);
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 16px;
  background: var(--surface);
  box-shadow: var(--shadow-pop);
  z-index: 130;
}

.task-settings-enter-active,
.task-settings-leave-active {
  transform-origin: top right;
  transition: opacity 0.14s ease, transform 0.14s ease;
}

.task-settings-enter-from,
.task-settings-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.985);
}

.task-settings__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex: none;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-soft);
  background: var(--panel-head-bg, var(--surface-sunken));
}

.task-settings__head strong {
  color: var(--text);
  font-size: 18px;
}

.task-settings__head button {
  width: 30px;
  height: 30px;
  padding: 0;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--text-muted);
  font-size: 25px;
  line-height: 1;
  cursor: pointer;
}

.task-settings__head button:hover {
  background: var(--surface-hover);
  color: var(--text);
}

.task-settings__body {
  flex: 1;
  min-height: 0;
  padding: 18px;
  overflow-y: auto;
}

.task-settings__grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

.task-settings__item {
  min-width: 0;
  min-height: 78px;
  padding: 15px 16px;
  border: 1px solid var(--border-soft);
  border-radius: 12px;
  background: var(--surface-sunken);
}

.task-settings__item--wide {
  grid-column: 1 / -1;
}

.task-settings__item--stepper,
.task-settings__field {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.task-settings__label {
  display: grid;
  gap: 5px;
  min-width: 0;
}

.task-settings__label strong {
  color: var(--text);
  font-size: 15px;
  font-weight: 650;
  white-space: nowrap;
}

.task-settings__label small {
  overflow: hidden;
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-settings__stepper {
  display: grid;
  grid-template-columns: 38px 44px 38px;
  align-items: center;
  flex: none;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
}

.task-settings__stepper button {
  height: 38px;
  border: 0;
  background: transparent;
  color: var(--text);
  font-size: 20px;
  cursor: pointer;
}

.task-settings__stepper button:hover:not(:disabled) {
  background: var(--surface-hover);
}

.task-settings__stepper button:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.task-settings__stepper span {
  color: var(--text);
  font-size: 15px;
  font-weight: 700;
  text-align: center;
}

:global(.upload-task-panel.is-expanded) .task-settings {
  max-height: calc(100vh - 64px);
}

@media (max-width: 768px) {
  .task-settings {
    max-height: calc(100vh - 78px);
  }

  .task-settings__grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 520px) {
  .task-settings {
    max-width: calc(100vw - 20px);
  }

  .task-settings__head {
    padding: 14px 16px;
  }

  .task-settings__body {
    padding: 12px;
  }

  .task-settings__item {
    padding: 13px;
  }
}
</style>
