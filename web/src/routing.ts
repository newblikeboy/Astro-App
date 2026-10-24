import { useSyncExternalStore } from "react";
import type { MouseEvent } from "react";

const eventName = "m4y:navigation";
/** subscribe subscribes the route store to browser navigation and application navigation events. */
function subscribe(listener: () => void) {
  window.addEventListener("popstate", listener);
  window.addEventListener("hashchange", listener);
  window.addEventListener(eventName, listener);
  return () => {
    window.removeEventListener("popstate", listener);
    window.removeEventListener("hashchange", listener);
    window.removeEventListener(eventName, listener);
  };
}
/** useAppLocation returns the current application URL through React's external-store subscription. */
export function useAppLocation() {
  const value = useSyncExternalStore(
    subscribe,
    () => location.pathname + location.search + location.hash,
  );
  const url = new URL(value, location.origin);
  url.pathname = url.pathname.replace(/\/+$/, "") || "/";
  return url;
}
/** navigate updates browser history and notifies route subscribers without a full page reload. */
export function navigate(path: string, replace = false) {
  const target = new URL(path, location.origin);
  if (target.origin !== location.origin)
    throw new Error("Navigation must stay on this website");
  const next = target.pathname + target.search + target.hash;
  if (next === location.pathname + location.search + location.hash) return;
  history[replace ? "replaceState" : "pushState"](null, "", next);
  window.dispatchEvent(new Event(eventName));
  window.scrollTo({ top: 0, behavior: "instant" });
}
/** followRoute intercepts an ordinary local-link click while preserving modified-click browser behavior. */
export function followRoute(
  event: MouseEvent<HTMLAnchorElement>,
  path: string,
) {
  if (
    event.button ||
    event.metaKey ||
    event.ctrlKey ||
    event.shiftKey ||
    event.altKey
  )
    return;
  event.preventDefault();
  navigate(path);
}
// Customer authentication never redirects to an external site or an admin route.
export function customerReturn(value: string | null) {
  if (value && /^(?:\/|\/kundli|\/account\/reports\/[A-Za-z0-9_-]+)\?video=[a-z0-9]+(?:-[a-z0-9]+)*$/.test(value) && value.split("?video=")[1].length >= 3 && value.split("?video=")[1].length <= 64) return value;
  if (
    value === "/" ||
    value === "/kundli" ||
    value === "/account" ||
    value === "/kundli-matching" ||
    value === "/shop" ||
    value === "/shop/bag" ||
    value === "/shop/orders"
  )
    return value;
  if (value && /^\/account\/reports\/[A-Za-z0-9_-]+$/.test(value)) return value;
  if (value && /^\/account\?tab=(bookings|orders|privacy)$/.test(value)) return value;
  if (value && /^\/shop\/stones\/[a-z0-9]+(?:-[a-z0-9]+)*$/.test(value))
    return value;
  return "/account";
}
/** loginPath builds the customer login URL with a validated local return destination. */
export function loginPath(next = "/account") {
  return "/login?next=" + encodeURIComponent(customerReturn(next));
}
