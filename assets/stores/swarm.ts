import { acceptHMRUpdate, defineStore } from "pinia";

import { Container, GroupedContainers } from "@/models/Container";
import { Service, Stack } from "@/models/Stack";

export const useSwarmStore = defineStore("swarm", () => {
  const containerStore = useContainerStore();
  const { containers } = storeToRefs(containerStore) as unknown as { containers: Ref<Container[]> };

  const runningContainers = computed(() => containers.value.filter((c) => c.state === "running"));

  const stacks = computed(() => {
    const namespaced: Record<string, Container[]> = {};
    for (const item of runningContainers.value) {
      const namespace = item.namespace;
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

      if (newServices.length === 0) continue;

      newStacks.push(
        new Stack(
          name,
          containers,
          newServices.sort((a, b) => a.name.localeCompare(b.name)),
        ),
      );
    }
    return newStacks.sort((a, b) => a.name.localeCompare(b.name));
  });

  const services = computed(() => {
    const services: Record<string, Container[]> = {};

    for (const container of runningContainers.value) {
      const service = container.labels["com.docker.swarm.service.name"];
      const namespace = container.namespace;

      if (service === undefined) continue;
      if (namespace) continue; // skip containers that already have a stack
      services[service] ||= [];
      services[service].push(container);
    }

    const serviceWithStack = stacks.value.flatMap((stack) => stack.services);

    const servicesWithoutStack = Object.entries(services).map(([name, containers]) => new Service(name, containers));

    return [...serviceWithStack, ...servicesWithoutStack].sort((a, b) => a.name.localeCompare(b.name));
  });

  const customGroups = computed(() => {
    const grouped: Record<string, Container[]> = {};

    for (const container of runningContainers.value) {
      const group = container.customGroup;
      if (group === undefined) continue;
      grouped[group] ||= [];
      grouped[group].push(container);
    }

    return Object.entries(grouped).map(([name, containers]) => new GroupedContainers(name, containers));
  });

  return {
    stacks,
    services,
    customGroups,
  };
});

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useSwarmStore, import.meta.hot));
}
