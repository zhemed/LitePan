<script setup lang="ts">
import { defineAsyncComponent, ref } from "vue";
import AppButton from "@/components/base/AppButton.vue";
import SectionTabBar from "@/components/admin/SectionTabBar.vue";
import { useSectionTabRoute } from "@/composables/useSectionTabRoute";
import "@/styles/admin-shared.css";

const AUTOMATION_TAB = "automation";
const tabs = [{ key: AUTOMATION_TAB, label: "自动联动" }];

const automationPanelRef = ref<{ openCreate: () => void } | null>(null);

const AutomationPanel = defineAsyncComponent(() => import("@/components/admin/AutomationPanel.vue"));

const { activeTab, setActiveTab } = useSectionTabRoute(AUTOMATION_TAB, [AUTOMATION_TAB]);
</script>

<template>
  <div class="settings">
    <SectionTabBar :model-value="activeTab" :tabs="tabs" @update:model-value="setActiveTab">
      <template #actions>
        <AppButton type="button" variant="primary" @click="automationPanelRef?.openCreate()">
          新增联动
        </AppButton>
      </template>
    </SectionTabBar>

    <AutomationPanel ref="automationPanelRef" />
  </div>
</template>

<style scoped>
.settings {
  padding-bottom: 24px;
}
</style>
