import { acceptHMRUpdate, defineStore } from "pinia";
import { Ref } from "vue";

export const usePinnedLogsStore = defineStore("pinnedLogs", () => {
  const containerStore = useContainerStore();
  const { allContainersById } = storeToRefs(containerStore);
  const pinnedContainerIds: Ref<string[]> = ref([]);

  const pinnedLogs = computed(() => pinnedContainerIds.value.map((id) => allContainersById.value[id]));

  const pinContainer = ({ id }: { id: string }) => pinnedContainerIds.value.push(id);
  const unPinContainer = ({ id }: { id: string }) =>
    pinnedContainerIds.value.splice(pinnedContainerIds.value.indexOf(id), 1);

  const isPinned = ({ id }: { id: string }) => pinnedContainerIds.value.includes(id);

  return {
    pinnedLogs,
    isPinned,
    pinContainer,
    unPinContainer,
  };
});

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(usePinnedLogsStore, import.meta.hot));
}
