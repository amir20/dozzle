/** Scroll position past which scrolling down compacts the bar, in px. */
export const COMPACT_OFFSET_PX = 80;
/** Scroll movements smaller than this are ignored, so a resting finger doesn't flicker the bar. */
export const SCROLL_JITTER_PX = 8;

export interface ScrollSample {
  /** `window.scrollY` now. */
  y: number;
  /** The last position a decision was made at. */
  lastY: number;
  /** `scrollHeight - innerHeight`: the furthest a real scroll can go. */
  maxY: number;
}

/**
 * Whether the bar should be compact after a scroll, or `null` to change nothing and keep
 * `lastY` where it is.
 *
 * Positions outside 0…maxY are iOS rubber-banding, whose bounce back would read as a scroll
 * in the other direction. Movements under the jitter threshold are held back rather than
 * dropped, so a slow scroll still adds up.
 */
export function nextCompact(
  { y, lastY, maxY }: ScrollSample,
  offset = COMPACT_OFFSET_PX,
  jitter = SCROLL_JITTER_PX,
): boolean | null {
  if (y < 0 || y > maxY) return null;
  const delta = y - lastY;
  if (Math.abs(delta) < jitter) return null;
  return delta > 0 && y > offset;
}

/**
 * True while the window is being scrolled down past the first screenful, false once it
 * scrolls back up. A page that pins itself to the viewport never scrolls the window, and
 * there it simply stays false.
 */
export function useCompactOnScroll() {
  const compact = ref(false);
  let lastY = 0;

  const onScroll = () => {
    const y = window.scrollY;
    const next = nextCompact({ y, lastY, maxY: document.documentElement.scrollHeight - window.innerHeight });
    if (next === null) return;
    lastY = y;
    compact.value = next;
  };

  onMounted(() => {
    lastY = window.scrollY;
    window.addEventListener("scroll", onScroll, { passive: true });
  });
  onBeforeUnmount(() => window.removeEventListener("scroll", onScroll));

  return compact;
}
