import { acceptHMRUpdate, defineStore } from "pinia";

import { Container } from "@/models/Container";
import { Service, Stack } from "@/models/Stack";

export const useSwarmStore = defineStore("stack", () => {
  const containerStore = useContainerStore();
  const { containers } = storeToRefs(containerStore) as unknown as { containers: Ref<Container[]> };

  const services = ref<Service[]>([]);

  const stacks = computed(() => {
    const runningContainers = containers.value.filter((container) => container.state === "running");
    const namespaced: Record<string, Container[]> = {};
    for (const item of runningContainers) {
      const namespace = item.labels["com.docker.stack.namespace"] ?? item.labels["com.docker.compose.project"];
      if (namespace === undefined) continue;
      namespaced[namespace] ||= [];
      namespaced[namespace].push(item);
    }

    const newStacks: Stack[] = [];

    for (const [name, containers] of Object.entries(namespaced)) {
      const services: Record<string, Container[]> = {};

      for (const container of containers) {
        const service = container.labels["com.docker.swarm.service.name"];

        if (service === undefined) continue;
        services[service] ||= [];
        services[service].push(container);
      }

      const newServices: Service[] = [];

      for (const [name, containers] of Object.entries(services)) {
        newServices.push(new Service(name, containers));
      }

      newStacks.push(new Stack(name, containers, newServices));
    }
    return newStacks;
  });

  return {
    stacks,
    services,
  };
});

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useSwarmStore, import.meta.hot));
}
