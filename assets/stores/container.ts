import { acceptHMRUpdate, defineStore } from "pinia";
import { ref, Ref, computed } from "vue";

import { showAllContainers } from "@/composables/settings";
import config from "@/stores/config";
import { Container } from "@/types/Container";

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
  es.addEventListener(
    "containers-changed",
    (e: Event) => (containers.value = JSON.parse((e as MessageEvent).data)),
    false
  );
  // es.addEventListener("container-stat", (e) => store.dispatch("UPDATE_STATS", JSON.parse(e.data)), false);
  // es.addEventListener("container-die", (e) => store.dispatch("UPDATE_CONTAINER", JSON.parse(e.data)), false);

  const currentContainer = (id: Ref<string>) => computed(() => allContainersById.value[id.value]);
  const appendActiveContainer = ({ id }: Container) => activeContainerIds.value.push(id);
  const removeActiveContainer = ({ id }: Container) => activeContainerIds.value.splice(activeContainerIds.value.indexOf(id), 1);
  return {
    containers,
    activeContainerIds,
    allContainersById,
    visibleContainers,
    activeContainers,
    currentContainer,
    appendActiveContainer,
    removeActiveContainer,
  };
});

// @ts-ignore
if (import.meta.hot) {
  // @ts-ignore
  import.meta.hot.accept(acceptHMRUpdate(useContainerStore, import.meta.hot));
}
