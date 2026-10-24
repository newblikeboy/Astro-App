import type { DevotionalCategory } from "./devotional-types";

export const devotionalPageSize = 12;

/** devotionalListPath creates shareable collection URLs; changing a category or search starts at page one. */
export function devotionalListPath(category: DevotionalCategory | undefined, query = "", page = 1) {
  const params = new URLSearchParams();
  if (query) params.set("q", query);
  if (page > 1) params.set("page", String(page));
  const suffix = params.toString();
  return `/bhakti${category ? "/" + category : ""}${suffix ? "?" + suffix : ""}`;
}

/** devotionalPage reads a bounded positive page number and clamps it to the available result pages. */
export function devotionalPage(value: string | null, totalPages: number) {
  const parsed = value && /^[1-9]\d{0,5}$/.test(value) ? Number(value) : 1;
  return Math.max(1, Math.min(parsed, totalPages));
}
