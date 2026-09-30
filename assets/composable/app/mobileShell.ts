import type { RouteLocationNormalizedLoaded, Router } from "vue-router";

/**
 * Routes that render a log stream. On a phone these are the pushed "detail" screens:
 * the tab bar steps aside so the bottom of the stream stays visible, and the header
 * carries a back button instead.
 */
const LOG_ROUTES = new Set<string>([
  "/container/[id]",
  "/container/[id].time.[datetime]",
  "/merged/[ids]",
  "/host/[id]",
  "/host-group/[name]",
  "/service/[name]",
  "/stack/[name]",
  "/group/[name]",
  "/namespace/[name]",
  "/owner/[name]",
]);

export function isLogRoute(route: RouteLocationNormalizedLoaded) {
  return typeof route.name === "string" && LOG_ROUTES.has(route.name);
}

/** Back to wherever the reader came from, or home when the view was opened cold from a link. */
export function goBack(router: Router) {
  if (window.history.state?.back) router.back();
  else router.push({ name: "/" });
}

/**
 * Safari gives a page swipe-from-the-edge to go back for free, but an app launched from
 * the home screen has no browser around it and loses the gesture. `navigator.standalone`
 * only exists on iOS, which is the point: Android's system back gesture already works
 * in a standalone app, and handling it here too would go back twice.
 */
export function useEdgeSwipeBack(enabled: Ref<boolean>) {
  const router = useRouter();
  const standalone = (navigator as Navigator & { standalone?: boolean }).standalone === true;
  if (!standalone) return;

  const EDGE = 24;
  const DISTANCE = 80;
  let start: { x: number; y: number } | undefined;

  useEventListener(
    window,
    "touchstart",
    (e: TouchEvent) => {
      const touch = e.touches[0];
      start =
        enabled.value && e.touches.length === 1 && touch.clientX <= EDGE
          ? { x: touch.clientX, y: touch.clientY }
          : undefined;
    },
    { passive: true },
  );

  useEventListener(
    window,
    "touchend",
    (e: TouchEvent) => {
      if (!start) return;
      const touch = e.changedTouches[0];
      const dx = touch.clientX - start.x;
      const dy = Math.abs(touch.clientY - start.y);
      start = undefined;
      // Mostly sideways, so a vertical scroll that began near the edge is left alone.
      if (dx > DISTANCE && dy < dx / 2) goBack(router);
    },
    { passive: true },
  );

  useEventListener(window, "touchcancel", () => (start = undefined), { passive: true });
}
