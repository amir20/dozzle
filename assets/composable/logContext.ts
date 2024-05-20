type LogContext = {
  streamConfig: { stdout: boolean; stderr: boolean };
};

const key = Symbol("loggingContext") as InjectionKey<LogContext>;

export const provideLoggingContext = () => {
  provide(key, {
    streamConfig: reactive({ stdout: true, stderr: true }),
  });
};

export const useLoggingContext = () => {
  const context = inject(key);
  if (!context) {
    throw new Error("No logging context provided");
  }
  return context;
};
