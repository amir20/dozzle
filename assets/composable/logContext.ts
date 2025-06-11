import { Container } from "@/models/Container";
import { Level } from "@/models/LogEntry";

type LogContext = {
  streamConfig: { stdout: boolean; stderr: boolean };
  containers: Container[];
  loadingMore: boolean;
  hasComplexLogs: boolean;
  levels: Set<Level>;
  showContainerName: boolean;
  showHostname: boolean;
  historical: boolean;
};

export const allLevels: Level[] = ["info", "debug", "warn", "error", "fatal", "trace", "unknown"];

// export for testing
export const loggingContextKey = Symbol("loggingContext") as InjectionKey<LogContext>;
const searchParams = new URLSearchParams(window.location.search);
const stdout = searchParams.has("stdout") ? searchParams.get("stdout") === "true" : true;
const stderr = searchParams.has("stderr") ? searchParams.get("stderr") === "true" : true;

export const provideLoggingContext = (
  containers: Ref<Container[]>,
  { showContainerName = false, showHostname = false, historical = false } = {},
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
    }),
  );

  return toRefs(context);
};
