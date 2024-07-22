import { Container } from "@/models/Container";

type LogContext = {
  streamConfig: { stdout: boolean; stderr: boolean };
  containers: Container[];
  loadingMore: boolean;
};

export const loggingContextKey = Symbol("loggingContext") as InjectionKey<LogContext>;

export const provideLoggingContext = (containers: Ref<Container[]>) => {
  provide(
    loggingContextKey,
    reactive({
      streamConfig: { stdout: true, stderr: true },
      containers,
      loadingMore: false,
    }),
  );
};

export const useLoggingContext = () => {
  const context = inject(loggingContextKey);
  if (!context) {
    throw new Error("No logging context provided");
  }
  return toRefs(context);
};
