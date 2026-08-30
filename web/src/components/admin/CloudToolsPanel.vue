<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue";
import ProxyToolsPanel from "@/components/admin/ProxyToolsPanel.vue";
import QuarkTVToolCard from "@/components/admin/QuarkTVToolCard.vue";
import AIToolCard from "@/components/admin/AIToolCard.vue";
import ClassificationToolCard from "@/components/admin/ClassificationToolCard.vue";
import CleanupToolCard from "@/components/admin/CleanupToolCard.vue";
import CoverExtractToolCard from "@/components/admin/CoverExtractToolCard.vue";
import LocalUploadToolCard from "@/components/admin/LocalUploadToolCard.vue";

const props = withDefaults(defineProps<{ searchOpen?: boolean }>(), { searchOpen: false });
const emit = defineEmits<{ "update:searchOpen": [boolean] }>();

const searchQuery = ref("");
const searchInputRef = ref<HTMLInputElement | null>(null);
const cardTitles = ["Emby 反代", "飞牛影视反代", "夸克 TV 接管", "AI 辅助识别", "目录整理分类", "从服务器上传", "垃圾清理工具", "视频海报生成"];

const hasMatch = computed(() => {
  const q = searchQuery.value.trim().toLowerCase();
  return !q || cardTitles.some((t) => t.toLowerCase().includes(q));
});

function closeSearch() {
  searchQuery.value = "";
  emit("update:searchOpen", false);
}

watch(
  () => props.searchOpen,
  async (open) => {
    if (open) {
      await nextTick();
      searchInputRef.value?.focus();
    } else {
      searchQuery.value = "";
    }
  },
);
</script>

<template>
  <div class="cloud-tools">
    <div v-if="searchOpen" class="tool-search">
      <div class="tool-search__mask" @click="closeSearch" />
      <div class="tool-search__box">
        <input ref="searchInputRef" v-model="searchQuery" placeholder="搜索工具，如：飞牛、Emby、反代" @keydown.esc="closeSearch" />
        <button type="button" @click="closeSearch">×</button>
      </div>
    </div>
    <div class="cloud-tools__grid">
      <ProxyToolsPanel :search-query="searchQuery" />

      <QuarkTVToolCard :search-query="searchQuery" />

      <AIToolCard :search-query="searchQuery" />

      <ClassificationToolCard :search-query="searchQuery" />

      <LocalUploadToolCard :search-query="searchQuery" />

      <CleanupToolCard :search-query="searchQuery" />

      <CoverExtractToolCard :search-query="searchQuery" />
    </div>
    <div v-if="searchOpen && !hasMatch" class="tool-search__empty">没有找到相关工具</div>
  </div>
</template>

<style scoped>
.tool-search__mask {
  position: fixed;
  inset: 0;
  z-index: var(--z-modal);
  background: rgba(15, 23, 42, 0.35);
}
.tool-search__box {
  position: fixed;
  top: 140px;
  left: 50%;
  transform: translateX(-50%);
  z-index: calc(var(--z-modal) + 1);
  display: flex;
  align-items: center;
  gap: 8px;
  width: min(520px, calc(100vw - 40px));
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-pop);
  padding: 12px 16px;
}
.tool-search__box input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  font-size: 15px;
  color: var(--text);
}
.tool-search__box button {
  border: none;
  background: transparent;
  color: var(--text-muted);
  font-size: 16px;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
}
.tool-search__box button:hover {
  background: var(--border-soft);
  color: var(--text);
}
.tool-search__empty {
  margin-top: 16px;
  text-align: center;
  color: var(--text-muted);
  font-size: 14px;
  padding: 40px 0;
}

.cloud-tools__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 300px), 1fr));
  align-items: start;
  gap: 16px;
}
</style>
