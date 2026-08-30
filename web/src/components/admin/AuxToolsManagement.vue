<script setup lang="ts">
import { computed, defineAsyncComponent, onDeactivated, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import AppButton from "@/components/base/AppButton.vue";
import SectionTabBar from "@/components/admin/SectionTabBar.vue";
import { useSectionTabRoute } from "@/composables/useSectionTabRoute";

const CloudToolsPanel = defineAsyncComponent(() => import("@/components/admin/CloudToolsPanel.vue"));
const BackupRestorePanel = defineAsyncComponent(() => import("@/components/admin/BackupRestorePanel.vue"));

const ENHANCED_TAB = "enhanced";
const BACKUP_TAB = "backup";
const tabs = [
  { key: ENHANCED_TAB, label: "增强工具" },
  { key: BACKUP_TAB, label: "备份管理" },
];

const enhancedSearchOpen = ref(false);

const route = useRoute();
const initialTab = String(route.query.tab ?? "") === BACKUP_TAB ? BACKUP_TAB : ENHANCED_TAB;
const { activeTab, setActiveTab } = useSectionTabRoute(initialTab, [ENHANCED_TAB, BACKUP_TAB]);

// 重面板首次访问时才挂载，之后只隐藏不销毁，保留已加载状态。
const tabsVisited = reactive<Record<string, boolean>>({});
watch(
  activeTab,
  (tab) => {
    tabsVisited[tab] = true;
    if (tab !== ENHANCED_TAB) enhancedSearchOpen.value = false;
  },
  { immediate: true },
);

const isEnhancedTab = computed(() => activeTab.value === ENHANCED_TAB);
const isBackupTab = computed(() => activeTab.value === BACKUP_TAB);

onDeactivated(() => {
  enhancedSearchOpen.value = false;
});
</script>

<template>
  <div class="settings">
    <SectionTabBar :model-value="activeTab" :tabs="tabs" @update:model-value="setActiveTab">
      <template #actions>
        <AppButton v-if="isEnhancedTab" type="button" variant="secondary" @click="enhancedSearchOpen = true">
          搜索工具
        </AppButton>
      </template>
    </SectionTabBar>

    <CloudToolsPanel
      v-if="tabsVisited[ENHANCED_TAB]"
      v-show="isEnhancedTab"
      :search-open="enhancedSearchOpen"
      @update:search-open="enhancedSearchOpen = $event"
    />
    <BackupRestorePanel v-if="tabsVisited[BACKUP_TAB]" v-show="isBackupTab" />
  </div>
</template>
