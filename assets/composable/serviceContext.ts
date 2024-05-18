import { Service } from "@/models/Stack";

type ServiceContext = {
  service: Ref<Service>;
  streamConfig: { stdout: boolean; stderr: boolean };
};

export const serviceContext = Symbol("stackContext") as InjectionKey<ServiceContext>;

export const provideServiceContext = (service: Ref<Service>) => {
  provide(serviceContext, {
    service,
    streamConfig: reactive({ stdout: true, stderr: true }),
  });
};

export const useServiceContext = () => {
  const context = inject(serviceContext);
  if (!context) {
    throw new Error("No service context provided");
  }
  return context;
};
