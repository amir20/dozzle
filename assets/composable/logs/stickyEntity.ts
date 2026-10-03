import type { Container } from "@/models/Container";

/**
 * Marks a container the view is holding on to as deleted once the store no longer
 * has it. A `destroy` event already does this, but the store also drops containers
 * that vanished with no event (removed while the tab slept), and those would stay
 * "running" forever: frozen stats in the totals, a Live chip, and Stop offered for
 * an id that no longer exists. Only checked against a complete list, so the moment
 * between a reconnect and its replay does not count as everything being gone.
 */
export function useMarkDropped(held: () => Container[]) {
  const { allContainersById, ready } = storeToRefs(useContainerStore());
  watch(
    () => [held(), allContainersById?.value, ready?.value] as const,
    ([containers, byId, isReady]) => {
      if (!isReady || !byId) return;
      for (const container of containers) {
        if (!byId[container.id] && container.state !== "deleted") container.state = "deleted";
      }
    },
    { immediate: true },
  );
}

/**
 * The entity a multi-container route shows, held on to while its members come and go.
 *
 * Groups, stacks, services, namespaces and owners are built from running containers,
 * so a stop, or the gap in a plain restart, makes the entity vanish for a moment.
 * Unmounting the view then would throw away its stream and scrollback, and the
 * stream would have picked the restarted container up on its own. So the last one
 * found is kept for as long as the route key stays the same. Only a key that never
 * resolved reads as missing.
 */
export function useStickyEntity<T>(source: () => T | undefined, key: () => string) {
  const last = shallowRef<{ key: string; value: T }>();
  watch(
    [source, key],
    ([value, k]) => {
      if (value) last.value = { key: k, value };
    },
    { immediate: true, flush: "sync" },
  );
  const entity = computed(() => {
    const value = source();
    if (value) return value;
    return last.value?.key === key() ? last.value.value : undefined;
  });
  // Every entity this holds is a set of containers.
  useMarkDropped(() => (entity.value as { containers?: Container[] } | undefined)?.containers ?? []);
  return entity;
}
