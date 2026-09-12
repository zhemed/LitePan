import { http } from "./client";
import type { Account } from "./types";

export interface PublicSystemConfig {
  /** 后端版本号（internal/buildinfo.Version），前端展示版本的唯一来源。 */
  version: string;
  index_account_switch_mode: "dropdown" | "floating";
  compact_home_enabled?: boolean;
  header_effects_enabled?: boolean;
}

export interface CacheHitRateResult {
  hit_rate: number;
}

export const publicApi = {
  listAccounts: () => http.get<Account[]>("/public/accounts"),
  systemConfig: () => http.get<PublicSystemConfig>("/public/system-config"),
  cacheHitRate: () => http.get<CacheHitRateResult>("/public/cache/hit-rate"),
};
