export interface TmdbHit {
  id?: number | string;
  title?: string;
  name?: string;
  original_title?: string;
  original_name?: string;
  release_date?: string;
  first_air_date?: string;
  media_type?: string;
  poster_path?: string;
}

export function hitTitle(hit: TmdbHit) {
  return hit.title || hit.name || hit.original_title || hit.original_name || "未命名";
}

export function hitYear(hit: TmdbHit): number | undefined {
  const raw = hit.release_date || hit.first_air_date || "";
  const y = Number(String(raw).slice(0, 4));
  return y > 1900 ? y : undefined;
}

export function hitId(hit: TmdbHit) {
  return String(hit.id ?? "");
}

export function hitMediaType(hit: TmdbHit, fallback = "movie") {
  const t = (hit.media_type || "").toLowerCase();
  if (t === "tv" || t === "movie") return t;
  if (hit.name || hit.first_air_date) return "tv";
  if (hit.title || hit.release_date) return "movie";
  return fallback;
}

export function hitKey(hit: TmdbHit) {
  return `${hitMediaType(hit)}:${hitId(hit)}`;
}

export function hitTypeLabel(hit: TmdbHit) {
  return hitMediaType(hit) === "tv" ? "剧集" : "电影";
}

export function hitPosterURL(hit: TmdbHit, size: "w154" | "w500" = "w154") {
  const path = (hit.poster_path || "").trim();
  if (!path) return "";
  if (path.startsWith("http://") || path.startsWith("https://")) return path;
  return `https://image.tmdb.org/t/p/${size}${path.startsWith("/") ? path : `/${path}`}`;
}
