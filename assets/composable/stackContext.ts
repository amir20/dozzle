import { Stack } from "@/models/Stack";

type StackContext = {
  stack: Ref<Stack | undefined>;
  streamConfig: { stdout: boolean; stderr: boolean };
};

export const stackContext = Symbol("stackContext") as InjectionKey<StackContext>;

export const provideStackContext = (stack: Ref<Stack | undefined>) => {
  provide(stackContext, {
    stack,
    streamConfig: reactive({ stdout: true, stderr: true }),
  });
};

export const useStackContext = () => {
  const context = inject(stackContext);
  if (!context) {
    throw new Error("No stack context provided");
  }
  return context;
};
