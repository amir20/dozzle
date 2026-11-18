import type { ViewContext } from "@/composable/logs/viewContext";

/**
 * What the cloud panels are about: the containers on screen, or the instance.
 *
 * A page about containers opens every panel scoped to them, and the chip at the
 * top of the drawer widens it to the whole instance. The widening belongs to
 * the containers it was made on, so navigating to another view starts scoped
 * again rather than carrying "all" to a page that never asked for it.
 */
const widenedFor = ref<string>();

export function useCloudScope() {
  const view = useViewContext();
  const store = useContainerStore();

  const inView = computed(() => view.value.containers);
  const key = computed(() => inView.value.map((c) => `${c.host}/${c.id}`).join(","));
  const widened = computed(() => key.value !== "" && widenedFor.value === key.value);

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
    widen: () => (widenedFor.value = key.value),
    narrow: () => (widenedFor.value = undefined),
  };
}
