import { Container } from "@/models/Container";

type ContainerContext = {
  container: Ref<Container>;
  streamConfig: { stdout: boolean; stderr: boolean };
};

export const containerContext = Symbol("containerContext") as InjectionKey<ContainerContext>;

export const provideContainerContext = (container: Ref<Container>) => {
  provide(containerContext, {
    container,
    streamConfig: reactive({ stdout: true, stderr: true }),
  });
};

export const useContainerContext = () => {
  const context = inject(containerContext);
  if (!context) {
    throw new Error("No container context provided");
  }
  return context;
};
