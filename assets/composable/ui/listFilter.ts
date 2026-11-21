import { computed, ref, watch } from "vue";

/**
 * A text filter over a grouped list. While a query is active every surviving group
 * renders open, and toggles land on a throwaway set, so the list's own collapsed
 * state is untouched and comes back as it was once the query is cleared.
 */
export function useListFilter() {
  const query = ref("");
  const needle = computed(() => query.value.trim().toLowerCase());
  const active = computed(() => needle.value !== "");

  const closedWhileFiltering = ref(new Set<string>());
  watch(active, (on) => {
    if (!on) closedWhileFiltering.value = new Set();
  });

  /** True when no query is set or any of `texts` contains it, ignoring case. */
  const matches = (...texts: string[]) => !active.value || texts.some((t) => t.toLowerCase().includes(needle.value));

  /**
   * Keeps whole groups whose name matches, and otherwise only their matching
   * children. Groups left with nothing are dropped.
   */
  const filterGroups = <G, C>(
    groups: G[],
    groupName: (g: G) => string | undefined,
    children: (g: G) => C[],
    childName: (c: C) => string,
    rebuild: (g: G, children: C[]) => G,
  ): G[] => {
    if (!active.value) return groups;
    const out: G[] = [];
    for (const group of groups) {
      const name = groupName(group);
      if (name !== undefined && matches(name)) {
        out.push(group);
        continue;
      }
      const kept = children(group).filter((c) => matches(childName(c)));
      if (kept.length > 0) out.push(rebuild(group, kept));
    }
    return out;
  };

  const isOpen = (key: string, persistedOpen: boolean) =>
    active.value ? !closedWhileFiltering.value.has(key) : persistedOpen;

  const setOpen = (key: string, open: boolean, persist: (open: boolean) => void) => {
    if (!active.value) return persist(open);
    const next = new Set(closedWhileFiltering.value);
    open ? next.delete(key) : next.add(key);
    closedWhileFiltering.value = next;
  };

  return { query, active, matches, filterGroups, isOpen, setOpen };
}
