import { defineStore } from "pinia";
import { ref } from "vue";
import { publicApi } from "@/api/public";

/**
 * 应用级信息 store。
 *
 * 版本号经 `GET /api/public/system-config` 运行期获取 —— 该端点是公共的（免鉴权），
 * 因此首页匿名访问也能正确显示版本。后端 `internal/buildinfo.Version` 是唯一真值，
 * 前端不保留任何版本字面量（含 fallback）。
 */
export const useAppInfoStore = defineStore("appInfo", () => {
  const version = ref("");

  // 进行中去重：首页 footer 与管理后台顶部可能同时挂载，共享同一次请求。
  let inflightLoad: Promise<void> | null = null;

  function load(): Promise<void> {
    if (version.value) return Promise.resolve();
    if (inflightLoad) return inflightLoad;
    inflightLoad = (async () => {
      try {
        version.value = (await publicApi.systemConfig()).version ?? "";
      } catch {
        // 取不到版本不影响页面可用性：保持空串，调用方据此隐藏版本展示。
        version.value = "";
      } finally {
        inflightLoad = null;
      }
    })();
    return inflightLoad;
  }

  return { version, load };
});
