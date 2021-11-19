import { acceptHMRUpdate, defineStore } from "pinia";
import { ref, computed } from "vue";

import { showAllContainers } from "@/composables/settings";
import config from "@/stores/config";

export interface Container {
  id: string;
  created: number;
  image: string;
  name: string;
  state: string;
  status: string;
  stat: ContainerStat;
}

export interface ContainerStat {
  cpu: number;
  memory: number;
  memoryUsage: number;
}

export const useContainerStore = defineStore("container", () => {
  const containers = ref<Container[]>([]);
  const activeContainerIds = ref<string[]>([]);

  const allContainersById = computed(() =>
    containers.value.reduce((acc, container) => {
      acc[container.id] = container;
      return acc;
    }, {} as Record<string, Container>)
  );

  const visibleContainers = computed(() => {
    const filter = showAllContainers.value ? () => true : (c: Container) => c.state === "running";
    return containers.value.filter(filter);
  });

  const activeContainers = computed(() => activeContainerIds.value.map((id) => allContainersById.value[id]));

  const es = new EventSource(`${config.base}/api/events/stream`);
  es.addEventListener("containers-changed", (e) => (containers.value = JSON.parse(e.data)), false);
  // es.addEventListener("container-stat", (e) => store.dispatch("UPDATE_STATS", JSON.parse(e.data)), false);
  // es.addEventListener("container-die", (e) => store.dispatch("UPDATE_CONTAINER", JSON.parse(e.data)), false);

  return {
    containers,
    activeContainerIds,
    allContainersById,
    visibleContainers,
    activeContainers,
  };
});

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useContainerStore, import.meta.hot));
}
