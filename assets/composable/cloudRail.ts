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
  const viewing = hasViewContext();

  // Every panel is about the logs on screen, so the rail belongs where there
  // are logs on screen. On the home page or in settings it would be a strip of
  // icons about nothing in particular.
  const available = computed(() => linked.value && viewing.value);

  /**
   * A phone has no room for a permanent strip beside the stream, and a 550px
   * column next to a 390px screen is not a column. There the rail is an overlay
   * instead: it has no resting state, opens over the logs when something asks
   * for a panel, and takes the screen with it.
   */
  const sheet = computed(() => available.value && isMobile.value);

  // Hidden is a preference, not a state: the strip is always there to bring
  // back, the way the nav collapses on the other edge. Closing the last panel
  // does not hide it, because then the icons would vanish on a click that was
  // about the panel. A sheet has nothing to collapse, so it is on screen for
  // exactly as long as a panel is open.
  const mounted = computed(() => {
    if (!available.value) return false;
    return sheet.value ? !!panel.value : !collapseCloudRail.value;
  });

  /** The tab on the edge that brings a collapsed rail back. Only the strip
   *  collapses, so a sheet never has one. */
  const collapsed = computed(() => available.value && !sheet.value && collapseCloudRail.value);

  // 550px is the width the assistant's paragraphs were written for. On a
  // smaller window the logs matter more than the panel, so it gives way.
  const panelWidth = computed(() => Math.min(550, Math.max(320, windowWidth.value * 0.45)));

  /** What the page has to keep clear on its right. Nothing under a sheet, which
   *  covers the stream rather than sitting beside it. */
  const railOffset = computed(() => {
    if (!mounted.value || sheet.value) return 0;
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
    // Same reasoning as openRail: ⇧⌘K asking for the assistant must not land on
    // a rail that was collapsed an hour ago and quietly do nothing.
    collapseCloudRail.value = false;
    panel.value = panel.value === next ? undefined : next;
  }

  return {
    panel,
    panelWidth,
    railOffset,
    sheet,
    mounted,
    collapsed,
    available,
    openRail,
    closeRail,
    toggleRail,
    hideRail,
    showRail,
  };
}
