import { acceptHMRUpdate, defineStore } from "pinia";

import { Container } from "@/models/Container";
import { Stack } from "@/models/Stack";

export const useStackStore = defineStore("stack", () => {
  const containerStore = useContainerStore();
  const { containers } = storeToRefs(containerStore) as unknown as { containers: Ref<Container[]> };

  const stacks = computed(() => {
    const namespaced: Record<string, Container[]> = {};

    for (const item of containers.value) {
      const namespace = item.labels["com.docker.stack.namespace"] ?? item.labels["com.docker.compose.project"];
      if (namespace === undefined) continue;
      namespaced[namespace] ||= [];
      namespaced[namespace].push(item);
    }

    return Object.entries(namespaced).map(([name, containers]) => new Stack(name, containers));
  });
  return {
    stacks,
  };
});

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useStackStore, import.meta.hot));
}
