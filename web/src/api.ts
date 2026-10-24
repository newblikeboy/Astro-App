import { t, getLanguage } from "./i18n";
import { formatPaise } from "./money";
export class APIError extends Error {
  /** constructor preserves the server's HTTP status alongside the localized request error message. */
  constructor(
    message: string,
    public readonly status: number,
  ) {
    super(message);
    this.name = "APIError";
  }
}
/** api sends same-origin JSON requests with cookies and optional CSRF, throwing APIError for failed responses. */
export async function api<T>(
  path: string,
  method = "GET",
  body?: unknown,
  csrf?: string,
): Promise<T> {
  const res = await fetch("/api" + path, {
    method,
    credentials: "same-origin",
    headers: {
      "Content-Type": "application/json",
      ...(csrf ? { "X-CSRF-Token": csrf } : {}),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const data = await res.json().catch(() => ({
    error: t("सर्वर से संपर्क नहीं हो पाया। फिर कोशिश करें।"),
  }));
  if (!res.ok)
    throw new APIError(t(data.error || "अनुरोध पूरा नहीं हुआ।"), res.status);
  return data as T;
}
/** money formats exact integer paise for display, returning an unavailable label for unsafe values. */
export const money = (p: number) =>
  Number.isSafeInteger(p) ? formatPaise(p) : t("Amount unavailable");
/** localTime formats an account timestamp in the browser's timezone without showing clock seconds. */
export const localTime = (s: string) =>
  s.startsWith("0001-")
    ? t("अभी")
    : new Intl.DateTimeFormat(getLanguage() === "hi" ? "hi-IN" : "en-IN", {
        dateStyle: "medium",
        timeStyle: "short",
        timeZone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      }).format(new Date(s));
export const zone = Intl.DateTimeFormat().resolvedOptions().timeZone;
/** track sends a consented analytics event with bounded campaign metadata; analytics failures stay non-blocking. */
export function track(name: string, product = "") {
  if (localStorage.getItem("m4y-analytics") !== "yes") return;
  const utm = JSON.parse(sessionStorage.getItem("m4y-utm") || "{}");
  void api("/events", "POST", { name, product, consent: true, ...utm }).catch(
    () => {},
  );
}
