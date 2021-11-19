import { acceptHMRUpdate, defineStore } from "pinia";
import { ref, computed } from "vue";

import { showAllContainers } from "@/composables/settings";

interface Container {
  id: string;
  name: string;
  state: string;
  stat: ContainerStat;
}

interface ContainerStat {
  cpu: number;
  memory: number;
  memoryUsage: number;
}

export const useContainersStore = defineStore("containers", () => {
  const containers = ref<Container[]>([]);
  const activeContainerIds = ref<string[]>([]);

  const allContainersById = computed(() =>
    containers.value.reduce((acc, container) => {
      acc[container.id] = container;
      return acc;
    }, {} as { [id: string]: Container })
  );

  const visibleContainers = computed(() => {
    const filter = showAllContainers.value ? () => true : (c: Container) => c.state === "running";
    return containers.value.filter(filter);
  });

  const activeContainers = computed(() => activeContainerIds.value.map((id) => allContainersById.value[id]));

  console.log("containers store created");

  return {
    containers,
    activeContainerIds,
    allContainersById,
    visibleContainers,
    activeContainers,
  };
});

if (import.meta.hot) import.meta.hot.accept(acceptHMRUpdate(useContainersStore, import.meta.hot));
