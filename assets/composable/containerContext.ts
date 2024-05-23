import { Container } from "@/models/Container";

type ContainerContext = {
  container: Ref<Container>;
};

export const containerContext = Symbol("containerContext") as InjectionKey<ContainerContext>;

export const provideContainerContext = (container: Ref<Container>) => {
  provide(containerContext, {
    container,
  });
};

export const useContainerContext = () => {
  const context = inject(containerContext);
  if (!context) {
    throw new Error("No container context provided");
  }
  return context;
};
