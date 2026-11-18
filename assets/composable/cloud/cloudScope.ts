import type { ViewContext } from "@/composable/logs/viewContext";

/**
 * What the cloud panels are about: the containers on screen, or the instance.
 *
 * A page about containers opens every panel scoped to them, and the chip at the
 * top of the drawer widens it to the whole instance. The widening belongs to
 * the page it was made on, so navigating to another view starts scoped
 * again rather than carrying "all" to a page that never asked for it.
 */
const widenedFor = ref<string>();
/** Whether the widening on screen was the bell's, made only to open its alerts. */
const widenedByBell = ref(false);

/** Undoes a widening the bell made, and nothing else: one the reader made with
 *  the chip stays until they narrow it. Called when the drawer closes. */
export function narrowBellWiden() {
  if (!widenedByBell.value) return;
  widenedByBell.value = false;
  widenedFor.value = undefined;
}

export function useCloudScope() {
  const view = useViewContext();
  const store = useContainerStore();

  const route = useRoute();

  const inView = computed(() => view.value.containers);
  // Keyed on the page, not on the containers it shows: a host or stack view
  // gains and loses containers as they start and stop, and that must not drop
  // the reader back to the narrow scope under them.
  const key = computed(() => route.path);
  const widened = computed(() => inView.value.length > 0 && widenedFor.value === key.value);

  /** True while the panels answer for the containers on screen. */
  const scoped = computed(() => inView.value.length > 0 && !widened.value);
  /** The containers the panels ask about. Empty means the instance. */
  const containers = computed(() => (scoped.value ? inView.value : []));
  /** The context a turn is asked with: the page itself, minus its containers once widened. */
  const scopedView = computed<ViewContext>(() =>
    widened.value ? { ...view.value, containers: [], hosts: [], lines: undefined } : view.value,
  );
  const total = computed(() => store.visibleContainers.length);

  return {
    inView,
    scoped,
    widened,
    containers,
    scopedView,
    total,
    /** `bell` marks a widening the drawer undoes on close. One already in place
     *  stays the reader's, so the bell never takes it back from them. */
    widen: (by?: "bell") => {
      if (by === "bell" && widened.value) return;
      widenedByBell.value = by === "bell";
      widenedFor.value = key.value;
    },
    narrow: () => {
      widenedByBell.value = false;
      widenedFor.value = undefined;
    },
  };
}
