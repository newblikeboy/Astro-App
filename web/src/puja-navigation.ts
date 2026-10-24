import type { PujaKind } from "./puja-types";

export const pujaPageSize = 12;

/** pujaListPath preserves a guide's originating filter, search and page while keeping category/search changes at page one by default. */
export function pujaListPath(kind?: PujaKind, query = "", page = 1) {
  const params = new URLSearchParams();
  if (kind) params.set("category", kind);
  if (query) params.set("q", query);
  if (page > 1) params.set("page", String(page));
  return "/puja-vidhi" + (params.size ? "?" + params.toString() : "");
}
