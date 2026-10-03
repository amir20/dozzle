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
  return computed(() => {
    const value = source();
    if (value) return value;
    return last.value?.key === key() ? last.value.value : undefined;
  });
}
