import { Container } from "@/models/Container";
import { Level } from "@/models/LogEntry";
import type { TimeRange } from "./timeRange";

type LogContext = {
  streamConfig: { stdout: boolean; stderr: boolean };
  containers: Container[];
  loadingMore: boolean;
  hasComplexLogs: boolean;
  levels: Set<Level>;
  showContainerName: boolean;
  showHostname: boolean;
  // Prefix every container in the view shares, dropped from the name tag.
  namePrefix: string;
  historical: boolean;
  // The slice of the log the view was opened on. Only a route's own view has
  // one; a pinned column is always plain live.
  timeRange: TimeRange;
};

export const allLevels: Level[] = ["info", "debug", "warn", "error", "fatal", "trace", "unknown"];

// export for testing
export const loggingContextKey = Symbol("loggingContext") as InjectionKey<LogContext>;
const searchParams = new URLSearchParams(window.location.search);
const stdout = searchParams.has("stdout") ? searchParams.get("stdout") === "true" : true;
const stderr = searchParams.has("stderr") ? searchParams.get("stderr") === "true" : true;

export const provideLoggingContext = (
  containers: Ref<Container[]>,
  {
    showContainerName = false,
    showHostname = false,
    historical = false,
    timeRange = { kind: "live" },
  }: {
    showContainerName?: boolean;
    showHostname?: boolean;
    historical?: boolean;
    timeRange?: MaybeRef<TimeRange>;
  } = {},
) => {
  provide(
    loggingContextKey,
    reactive({
      streamConfig: { stdout, stderr },
      containers,
      loadingMore: false,
      hasComplexLogs: false,
      levels: new Set<Level>(allLevels),
      showContainerName,
      showHostname,
      namePrefix: computed(() => sharedNamePrefix(containers.value)),
      historical,
      timeRange,
    }),
  );
};

/**
 * The `<project>-` or `<project>_` prefix to drop from container names in a merged
 * view, so `myapp-web-1` and `myapp-worker-1` read as `web-1` and `worker-1`. Only
 * when every container has the same compose/stack project and its name carries it.
 * The project is a fixed property of each container, so the prefix only changes if
 * a container from another project joins, never as lines stream in.
 */
export function sharedNamePrefix(containers: Container[]): string {
  if (containers.length < 2) return "";
  const namespace = containers[0].namespace;
  if (!namespace) return "";
  for (const separator of ["-", "_"]) {
    const prefix = namespace + separator;
    if (
      containers.every((c) => c.namespace === namespace && c.name.length > prefix.length && c.name.startsWith(prefix))
    )
      return prefix;
  }
  return "";
}

export const useLoggingContext = () => {
  const context = inject(
    loggingContextKey,
    reactive({
      streamConfig: { stdout: true, stderr: true },
      containers: [],
      loadingMore: false,
      hasComplexLogs: false,
      levels: new Set<Level>(allLevels),
      showContainerName: false,
      showHostname: false,
      namePrefix: "",
      historical: false,
      timeRange: { kind: "live" } as TimeRange,
    }),
  );

  return toRefs(context);
};

/**
 * The view's streams, search and levels as query params, what the server's logFilter
 * reads. Every request for this view's lines starts from these.
 */
export function useLogFilterParams() {
  const { streamConfig, levels } = useLoggingContext();
  const { appliedSearchFilter, inverseFilter } = useSearchFilter();
  return computed(() => {
    const params = new URLSearchParams();
    if (streamConfig.value.stdout) params.append("stdout", "1");
    if (streamConfig.value.stderr) params.append("stderr", "1");
    if (appliedSearchFilter.value) {
      params.append("filter", appliedSearchFilter.value);
      if (inverseFilter.value) params.append("inverse", "true");
    }
    for (const level of levels.value) params.append("levels", level);
    return params;
  });
}
