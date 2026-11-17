export type RailPanel = "chat" | "metrics" | "alerts";

/**
 * Which cloud panel the drawer is showing, if any.
 *
 * The layout opens the drawer when this is set and closes it when it clears,
 * so anything can open a panel by name from anywhere: a log row's menu,
 * the command palette, the container toolbar. State lives here rather than in
 * the component so navigating does not close what you opened.
 *
 * It exists only where cloud is linked, because every panel on it is memory:
 * an answer that read the history, the metrics beyond Dozzle's live window,
 * the alerts that survived a refresh. Nothing local is gated behind it.
 */
const panel = ref<RailPanel>();

/** The panel last opened, so a single "Cloud" entry point (a phone's toolbar
 *  menu, which has no room for three rows) lands where the reader left off. */
const lastPanel = ref<RailPanel>("chat");
watch(panel, (next) => {
  if (next) lastPanel.value = next;
  // Counted on the change, not on ChatPane mounting: the rail remounts with chat
  // still open when navigation leaves and returns to a log view.
  if (next === "chat") trackUsage("cloud.chat");
});

/**
 * The recent alerts, scoped to whatever log view is on screen.
 *
 * The rail's panel answers "what fired on what you're looking at" — the only
 * reason it earns a place beside the stream rather than a link to the
 * notifications page — so its bell has to ask the same question. An
 * instance-wide dot there promised rows the panel then filtered away.
 *
 * Lives beside the rail rather than with the alerts because the bell needs the
 * scope while the panel is closed and unmounted, and one definition keeps the
 * dot and the rows it promises from drifting apart again.
 */
export function useViewAlerts() {
  const { alerts, unseenIn } = useRecentAlerts();
  const view = useViewContext();

  const ids = computed(() => new Set(view.value.containers.map((c) => c.id)));
  const scoped = computed(() => ids.value.size > 0);
  const single = computed(() => ids.value.size === 1);

  // On a view with no containers of its own (the home page, settings) there is
  // nothing to scope to, so the panel shows the instance the way the page does.
  const visible = computed(() =>
    scoped.value ? alerts.value.filter((a) => ids.value.has(a.containerId)) : alerts.value,
  );

  return { visible, scoped, single, unseen: unseenIn(ids) };
}

export function useCloudRail() {
  const { linked } = useCloudSurface();
  const viewing = hasViewContext();

  // Every panel is about the logs on screen, so the drawer belongs where there
  // are logs on screen. On the home page or in settings it would be about
  // nothing in particular.
  const available = computed(() => linked.value && viewing.value);

  function openRail(next: RailPanel) {
    panel.value = next;
  }

  /** Opens the panel that most needs reading: alerts when something fired that
   *  has not been seen, otherwise wherever the reader was last. */
  function openCloud(unseenAlerts: boolean) {
    openRail(unseenAlerts ? "alerts" : lastPanel.value);
  }

  function closeRail() {
    panel.value = undefined;
  }

  function toggleRail(next: RailPanel) {
    panel.value = panel.value === next ? undefined : next;
  }

  return { panel, available, openRail, openCloud, closeRail, toggleRail };
}
