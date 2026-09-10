import { acceptHMRUpdate, defineStore } from "pinia";
import { Ref } from "vue";

export const usePinnedLogsStore = defineStore("pinnedLogs", () => {
  const containerStore = useContainerStore();
  const { allContainersById } = storeToRefs(containerStore);
  // Seeded from `?columns=` by usePinnedColumnsInUrl(), which the layout calls before it
  // renders the columns.
  const pinnedContainerIds: Ref<string[]> = ref([]);

  // An id stays pinned even while unknown: on a cold load the containers arrive over SSE after
  // the first render, so dropping it here would silently discard a shared link's columns.
  const pinnedLogs = computed(() =>
    pinnedContainerIds.value.map((id) => allContainersById.value[id]).filter((container) => container),
  );

  const pinContainer = ({ id }: { id: string }) => {
    if (!pinnedContainerIds.value.includes(id)) pinnedContainerIds.value.push(id);
  };

  const unPinContainer = ({ id }: { id: string }) => {
    const index = pinnedContainerIds.value.indexOf(id);
    if (index !== -1) pinnedContainerIds.value.splice(index, 1);
  };

  const isPinned = ({ id }: { id: string }) => pinnedContainerIds.value.includes(id);

  return {
    pinnedContainerIds,
    pinnedLogs,
    isPinned,
    pinContainer,
    unPinContainer,
  };
});

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(usePinnedLogsStore, import.meta.hot));
}
