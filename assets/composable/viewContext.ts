import stripAnsi from "strip-ansi";
import { allLevels } from "@/composable/logContext";
import {
  GroupedLogEntry,
  LoadMoreLogEntry,
  SkippedLogsEntry,
  type Level,
  type LogEntry,
  type LogMessage,
} from "@/models/LogEntry";

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
  /** The lines on screen, oldest first. */
  lines?: ViewLogLine[];
  /** The one line the user pointed at, when they asked from a log row. */
  focused?: ViewLogLine;
};

/** One line as Dozzle rendered it: ANSI stripped, message truncated. What the
 * user saw, rather than what the container wrote. */
export type ViewLogLine = {
  timestamp: string;
  level?: string;
  /** Only on a view that merges containers, where a line is ambiguous without it. */
  containerId?: string;
  message: string;
};

/** How much of the window travels with a turn.
 *
 * The buffer holds far more than these, but a turn is about what is in front of
 * the user, and the assistant can always fetch more with a tool. The budget is
 * the backstop: one container writing 4 KB JSON blobs would otherwise send a
 * megabyte of context nobody reads. */
const MAX_LINES = 150;
const MAX_LINE_CHARS = 2000;
const MAX_TOTAL_CHARS = 120_000;

export function toViewLogLine(entry: LogEntry<LogMessage>, withContainer = false): ViewLogLine {
  const text = entry instanceof GroupedLogEntry ? entry.message.join("\n") : entry.rawMessage || String(entry.message);
  const message = stripAnsi(text);

  return {
    timestamp: entry.date.toISOString(),
    level: entry.level === "unknown" ? undefined : entry.level,
    containerId: withContainer ? entry.containerID : undefined,
    message: message.length > MAX_LINE_CHARS ? message.slice(0, MAX_LINE_CHARS) + "…" : message,
  };
}

/** The tail of what is on screen, newest last, within the budget. Exported for testing. */
export function toViewLogLines(entries: LogEntry<LogMessage>[], withContainer: boolean): ViewLogLine[] {
  const lines: ViewLogLine[] = [];
  let budget = MAX_TOTAL_CHARS;

  // Walk backwards: when something has to be dropped it is the oldest line,
  // not the one the user is looking at.
  for (let i = entries.length - 1; i >= 0 && lines.length < MAX_LINES && budget > 0; i--) {
    const entry = entries[i];
    // Placeholders are chrome, not log: the assistant reading "load more" as a
    // line is worse than the gap it stands for.
    if (entry instanceof SkippedLogsEntry || entry instanceof LoadMoreLogEntry) continue;
    const line = toViewLogLine(entry, withContainer);
    budget -= line.message.length;
    lines.push(line);
  }

  return lines.reverse();
}

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

/**
 * The context of the viewer that is currently mounted.
 *
 * The chat pane and the command palette both live in the layout, outside the
 * viewer's provide/inject subtree, so injecting the logging and scroll contexts
 * there hands back the empty defaults: no containers, no search, a clock frozen
 * at the moment the fallback was created. The viewer publishes what it shows
 * here instead, and everyone else reads it.
 */
const published = shallowRef<ComputedRef<ViewContext>>();
const publishedLines = shallowRef<Ref<LogEntry<LogMessage>[]>>();

/** Builds the context from the contexts the viewer has in hand. Takes the
 * scroll context directly because the component that provides it cannot inject
 * its own. */
export function buildViewContext(scroll: { currentDate: Date }): ComputedRef<ViewContext> {
  const route = useRoute();
  const { containers, levels, historical } = useLoggingContext();
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
      visibleAt: scroll.currentDate.toISOString(),
      historical: historical.value,
    };
  });
}

/** Publishes this viewer as the one the assistant is looking at, for as long as
 * it stays mounted. */
export function publishViewContext(view: ComputedRef<ViewContext>) {
  published.value = view;
  onScopeDispose(() => {
    if (published.value === view) published.value = undefined;
  });
}

/** Whether this subtree is the primary viewer. A pinned column is a second view
 * of some other container, and publishing from it would make the assistant
 * follow the sidecar instead of the page. */
const ownerKey = Symbol("viewContextOwner") as InjectionKey<boolean>;

export const provideViewContextOwner = (owns: boolean) => provide(ownerKey, owns);
export const isViewContextOwner = () => inject(ownerKey, true);

/** Publishes the entries on screen, after filtering, so a turn can carry the
 * window the user is reading rather than making the assistant guess at it. */
export function publishVisibleLogs(entries: Ref<LogEntry<LogMessage>[]>) {
  publishedLines.value = entries;
  onScopeDispose(() => {
    if (publishedLines.value === entries) publishedLines.value = undefined;
  });
}

/** Whether a log viewer is on screen at all.
 *
 * True exactly where one has published, which is every log view and nothing
 * else: the home page, settings and notifications never mount one. Beats a list
 * of route names, which goes stale the moment a route is added. */
export function hasViewContext(): ComputedRef<boolean> {
  return computed(() => published.value !== undefined);
}

export function useViewContext(): ComputedRef<ViewContext> {
  const route = useRoute();

  return computed(() => {
    const view = published.value?.value ?? {
      kind: routeKind(route.name),
      target: typeof route.params.id === "string" ? route.params.id : undefined,
      containers: [],
      hosts: [],
      visibleAt: new Date().toISOString(),
      historical: false,
    };

    const entries = publishedLines.value?.value ?? [];
    if (entries.length === 0) return view;

    return { ...view, lines: toViewLogLines(entries, view.containers.length > 1) };
  });
}
