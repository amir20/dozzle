import { Container } from "@/models/Container";

type LogContext = {
  streamConfig: { stdout: boolean; stderr: boolean };
  containers: Ref<Container[]>;
};

export const loggingContextKey = Symbol("loggingContext") as InjectionKey<LogContext>;

export const provideLoggingContext = (containers: Ref<Container[]>) => {
  provide(loggingContextKey, {
    streamConfig: reactive({ stdout: true, stderr: true }),
    containers,
  });
};

export const useLoggingContext = () => {
  const context = inject(loggingContextKey);
  if (!context) {
    throw new Error("No logging context provided");
  }
  return context;
};
