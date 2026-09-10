import { allLevels } from "@/composable/logContext";
import type { Level } from "@/models/LogEntry";

/**
 * What the user is currently looking at, as one object.
 *
 * The whole point of asking the assistant from inside Dozzle is that nobody
 * should have to name the thing already on their screen. Every entry point
 * collects this and sends it with the turn, and the context chips in the chat
 * pane render from the same object, so what the assistant knows is exactly what
 * the user can see it knows.
 */
export type ViewContext = {
  /** Route shape: "container", "host", "merged", "stack", … "index" for home. */
  kind: string;
  /** The route's own id or name, when it has one. */
  target?: string;
  /** Containers currently in the viewer, in the order they are shown. */
  containers: { id: string; name: string; host: string }[];
  /** Hosts those containers live on, de-duplicated. */
  hosts: string[];
  /** The active search term, when the user is actually searching. */
  search?: string;
  /** Levels the user narrowed to. Absent when every level is shown, because
   * "all levels" is the default rather than a filter worth reporting. */
  levels?: Level[];
  /** The moment currently in view, ISO 8601. */
  visibleAt: string;
  /** True on the historical (time-travel) view rather than the live stream. */
  historical: boolean;
};

/**
 * Reduce a typed-router name to the shape of the page: "/container/[id]"
 * becomes "container", "/" becomes "index". Deriving it beats a hardcoded list
 * that silently goes stale when a route is added.
 */
export function routeKind(name: string | symbol | null | undefined): string {
  if (typeof name !== "string" || name === "" || name === "/") return "index";
  const [first] = name.replace(/^\//, "").split("/");
  // File-based routes name their params in brackets: "[id].time.[datetime]".
  return first.replace(/\..*$/, "").replace(/[[\]]/g, "") || "index";
}

/** Levels worth reporting: absent unless the user narrowed the set. */
export function narrowedLevels(levels: Set<Level>): Level[] | undefined {
  if (levels.size === 0 || levels.size >= allLevels.length) return undefined;
  return allLevels.filter((level) => levels.has(level));
}

export function useViewContext(): ComputedRef<ViewContext> {
  const route = useRoute();
  const { containers, levels, historical } = useLoggingContext();
  const { currentDate } = useScrollContext();
  const { debouncedSearchFilter, isSearching } = useSearchFilter();

  return computed(() => {
    const visible = containers.value ?? [];
    return {
      kind: routeKind(route.name),
      target: typeof route.params.id === "string" ? route.params.id : undefined,
      containers: visible.map(({ id, name, host }) => ({ id, name, host })),
      hosts: [...new Set(visible.map(({ host }) => host))],
      search: isSearching.value ? debouncedSearchFilter.value : undefined,
      levels: narrowedLevels(levels.value),
      visibleAt: currentDate.value.toISOString(),
      historical: historical.value,
    };
  });
}
