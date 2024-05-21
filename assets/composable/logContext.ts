import { Container } from "@/models/Container";

type LogContext = {
  streamConfig: { stdout: boolean; stderr: boolean };
  containers: Ref<Container[]>;
};

const key = Symbol("loggingContext") as InjectionKey<LogContext>;

export const provideLoggingContext = (containers: Ref<Container[]>) => {
  provide(key, {
    streamConfig: reactive({ stdout: true, stderr: true }),
    containers,
  });
};

export const useLoggingContext = () => {
  const context = inject(key);
  if (!context) {
    throw new Error("No logging context provided");
  }
  return context;
};
