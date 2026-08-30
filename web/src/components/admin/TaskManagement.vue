<script setup lang="ts">
import { computed, defineAsyncComponent, nextTick, onActivated, onDeactivated, onUnmounted, reactive, ref, watch, watchEffect } from "vue";
import AppButton from "@/components/base/AppButton.vue";
import AdminTaskTabHeader from "@/components/admin/AdminTaskTabHeader.vue";
import type { AdminTaskTabStat } from "@/components/admin/adminTaskTabHeader";
import SectionTabBar from "@/components/admin/SectionTabBar.vue";
import CacheRuntimeStats from "@/components/admin/CacheRuntimeStats.vue";
import AdminSettingsDrawer from "@/components/admin/AdminSettingsDrawer.vue";
import { useSectionTabRoute } from "@/composables/useSectionTabRoute";
import { useSettingsPageDirty } from "@/composables/useSettingsPageDirty";
import { readPanelSaving, type SettingsPanelExpose } from "@/composables/useSettingsForm";
import type CacheRetentionPanelComponent from "@/components/admin/CacheRetentionPanel.vue";
import type MediaOrganizePanelComponent from "@/components/admin/MediaOrganizePanel.vue";
import "@/styles/admin-shared.css";

const CACHE_TAB = "cache";
const ORGANIZE_TAB = "organize";
const AUTOMATION_TAB = "automation";
const tabs = [
  { key: CACHE_TAB, label: "缓存任务" },
  { key: ORGANIZE_TAB, label: "目录整理" },
  { key: AUTOMATION_TAB, label: "自动联动" },
];

type DrawerKind = "organize" | "cache";

const settingsDrawerOpen = ref(false);
const drawerKind = ref<DrawerKind>("cache");
const drawerKindsVisited = reactive<Record<DrawerKind, boolean>>({
  organize: false,
  cache: false,
});

const retentionPanelRef = ref<InstanceType<typeof CacheRetentionPanelComponent> | null>(null);
const cacheSettingsRef = ref<SettingsPanelExpose | null>(null);
const organizePanelRef = ref<InstanceType<typeof MediaOrganizePanelComponent> | null>(null);
const automationPanelRef = ref<{ openCreate: () => void } | null>(null);
const organizeSettingsRef = ref<SettingsPanelExpose | null>(null);

const CacheRetentionPanel = defineAsyncComponent(() => import("@/components/admin/CacheRetentionPanel.vue"));
const CacheSettingsPanel = defineAsyncComponent(() => import("@/components/admin/CacheSettingsPanel.vue"));
const AutomationPanel = defineAsyncComponent(() => import("@/components/admin/AutomationPanel.vue"));
const MediaOrganizePanel = defineAsyncComponent(() => import("@/components/admin/MediaOrganizePanel.vue"));
const MediaOrganizeSettings = defineAsyncComponent(() => import("@/components/admin/MediaOrganizeSettings.vue"));

const cachePanelDirty = ref(false);
const organizePanelDirty = ref(false);

watchEffect(() => {
  cachePanelDirty.value = (cacheSettingsRef.value as SettingsPanelExpose | null)?.isDirty?.() ?? false;
});

watchEffect(() => {
  organizePanelDirty.value = (organizeSettingsRef.value as SettingsPanelExpose | null)?.isDirty?.() ?? false;
});

const drawerDirty = computed(() => {
  if (drawerKind.value === "cache") return cachePanelDirty.value;
  return organizePanelDirty.value;
});

const settingsPageDirty = computed(() => settingsDrawerOpen.value && drawerDirty.value);

function revertDrawerSettings() {
  if (drawerKind.value === "cache") cacheSettingsRef.value?.revert?.();
  else organizeSettingsRef.value?.revert?.();
}

const { confirmDiscardChanges } = useSettingsPageDirty(settingsPageDirty, revertDrawerSettings);

const { activeTab, setActiveTab } = useSectionTabRoute(
  CACHE_TAB,
  [CACHE_TAB, ORGANIZE_TAB, AUTOMATION_TAB],
  {
    beforeTabChange: async (_from, _to) => {
      if (!settingsDrawerOpen.value) return true;
      const ok = await confirmDiscardChanges(() => drawerDirty.value);
      if (!ok) return false;
      settingsDrawerOpen.value = false;
      return true;
    },
  },
);

const tabsVisited = reactive<Record<string, boolean>>({});
watch(activeTab, (tab) => {
  tabsVisited[tab] = true;
}, { immediate: true });

const organizeTabStats = computed<AdminTaskTabStat[]>(() => [
  { icon: "fa-list", value: organizePanelRef.value?.taskCount ?? 0, label: "任务数量", tone: "blue" },
  { icon: "fa-play", value: organizePanelRef.value?.runningCount ?? 0, label: "执行中", tone: "purple" },
  {
    icon: "fa-pause",
    value: organizePanelRef.value?.errorTaskCount ?? 0,
    label: "有失败",
    tone: "amber",
  },
]);

const drawerTitle = computed(() => {
  if (drawerKind.value === "organize") return "整理设置";
  return "缓存设置";
});
const drawerSaving = computed(() => {
  if (drawerKind.value === "cache") return readPanelSaving(cacheSettingsRef.value?.saving);
  return readPanelSaving(organizeSettingsRef.value?.saving);
});
const drawerCanSave = drawerDirty;

async function openSettingsDrawer(kind?: DrawerKind) {
  drawerKind.value =
    kind ??
    (activeTab.value === ORGANIZE_TAB
      ? "organize"
      : "cache");
  drawerKindsVisited[drawerKind.value] = true;
  settingsDrawerOpen.value = true;
  if (drawerKind.value === "organize") {
    await nextTick();
    if (organizeSettingsRef.value && !organizeSettingsRef.value.isDirty?.()) {
      await organizeSettingsRef.value.reload?.();
    }
  } else {
    await nextTick();
    if (cacheSettingsRef.value && !cacheSettingsRef.value.isDirty?.()) {
      await cacheSettingsRef.value.reload?.();
    }
  }
}

async function closeSettingsDrawer() {
  if (!(await confirmDiscardChanges(() => drawerDirty.value))) return;
  settingsDrawerOpen.value = false;
}

async function handleDrawerSave() {
  if (drawerKind.value === "cache") await cacheSettingsRef.value?.save?.();
  else await organizeSettingsRef.value?.save?.();
}

function activatePage() {}

function deactivatePage() {
  if (settingsDrawerOpen.value) {
    if (drawerDirty.value) revertDrawerSettings();
    settingsDrawerOpen.value = false;
  }
}

onActivated(activatePage);
onDeactivated(deactivatePage);
onUnmounted(deactivatePage);
</script>

<template>
  <div class="settings">
    <SectionTabBar :model-value="activeTab" :tabs="tabs" @update:model-value="setActiveTab">
      <template #actions>
        <AppButton
          v-if="activeTab === CACHE_TAB"
          type="button"
          variant="primary"
          @click="retentionPanelRef?.openCreate()"
        >
          添加任务
        </AppButton>
        <AppButton
          v-else-if="activeTab === ORGANIZE_TAB"
          type="button"
          variant="primary"
          @click="organizePanelRef?.openCreate()"
        >
          添加任务
        </AppButton>
        <AppButton
          v-else-if="activeTab === AUTOMATION_TAB"
          type="button"
          variant="primary"
          @click="automationPanelRef?.openCreate()"
        >
          新增联动
        </AppButton>
      </template>
    </SectionTabBar>

    <div v-if="tabsVisited[CACHE_TAB]" v-show="activeTab === CACHE_TAB">
      <AdminTaskTabHeader
        settings-title="缓存设置"
        settings-hint="通用缓存 · WebDAV"
        @open-settings="openSettingsDrawer('cache')"
      >
        <CacheRuntimeStats />
      </AdminTaskTabHeader>
      <CacheRetentionPanel ref="retentionPanelRef" hide-stats />
    </div>

    <div v-if="tabsVisited[ORGANIZE_TAB]" v-show="activeTab === ORGANIZE_TAB">
      <AdminTaskTabHeader
        :stats="organizeTabStats"
        :refreshing="organizePanelRef?.refreshing ?? false"
        settings-title="整理设置"
        settings-hint="TMDB · 命名规则"
        @refresh="organizePanelRef?.loadTasks()"
        @open-settings="openSettingsDrawer('organize')"
      />
      <MediaOrganizePanel ref="organizePanelRef" hide-stats />
    </div>

    <div v-if="tabsVisited[AUTOMATION_TAB]" v-show="activeTab === AUTOMATION_TAB">
      <AutomationPanel ref="automationPanelRef" />
    </div>

    <AdminSettingsDrawer
      :open="settingsDrawerOpen"
      :title="drawerTitle"
      :saving="drawerSaving"
      :can-save="drawerCanSave"
      @close="closeSettingsDrawer"
      @cancel="closeSettingsDrawer"
      @save="handleDrawerSave"
    >
      <CacheSettingsPanel v-if="drawerKindsVisited.cache" v-show="drawerKind === 'cache'" ref="cacheSettingsRef" />
      <MediaOrganizeSettings v-if="drawerKindsVisited.organize" v-show="drawerKind === 'organize'" ref="organizeSettingsRef" />
    </AdminSettingsDrawer>
  </div>
</template>

<style scoped>
.settings {
  padding-bottom: 24px;
}
</style>
