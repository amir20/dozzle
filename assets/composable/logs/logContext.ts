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
      historical,
      timeRange,
    }),
  );
};

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
