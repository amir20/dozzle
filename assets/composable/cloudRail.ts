import { collapseCloudRail } from "@/stores/settings";

export type RailPanel = "chat" | "metrics" | "alerts";

/**
 * The rail beside the stream, and which of its panels is open.
 *
 * Mounted once in the layout, opened by name from anywhere: a log row's menu,
 * the command palette, the container toolbar. State lives here rather than in
 * the component so navigating does not close what you opened.
 *
 * It exists only where cloud is linked, because every panel on it is memory:
 * an answer that read the history, the metrics beyond Dozzle's live window,
 * the alerts that survived a refresh. Nothing local is gated behind it.
 */
const panel = ref<RailPanel>();

/** The icon strip's width. Shared with the layout, which pads the page by it so
 *  the rail never sits over the logs. */
export const RAIL_WIDTH = 48;

export function useCloudRail() {
  const { width: windowWidth } = useWindowSize();
  const { linked } = useCloudSurface();

  // Hidden is a preference, not a state: the strip is always there to bring
  // back, the way the nav collapses on the other edge. Closing the last panel
  // does not hide it, because then the icons would vanish on a click that was
  // about the panel.
  const visible = computed(() => linked.value && !isMobile.value && !collapseCloudRail.value);

  // 550px is the width the assistant's paragraphs were written for. On a
  // smaller window the logs matter more than the panel, so it gives way.
  const panelWidth = computed(() => Math.min(550, Math.max(320, windowWidth.value * 0.45)));

  /** What the page has to keep clear on its right. */
  const railOffset = computed(() => {
    if (!visible.value) return 0;
    return RAIL_WIDTH + (panel.value ? panelWidth.value : 0);
  });

  function hideRail() {
    collapseCloudRail.value = true;
    panel.value = undefined;
  }

  function showRail() {
    collapseCloudRail.value = false;
  }

  function openRail(next: RailPanel) {
    // Asking for a panel is asking for the rail: an entry point elsewhere in
    // the app should not silently do nothing because it was collapsed.
    collapseCloudRail.value = false;
    panel.value = next;
  }

  function closeRail() {
    panel.value = undefined;
  }

  function toggleRail(next: RailPanel) {
    panel.value = panel.value === next ? undefined : next;
  }

  return { panel, panelWidth, railOffset, visible, openRail, closeRail, toggleRail, hideRail, showRail };
}
