import { Container } from "@/models/Container";

const DOZZLE_HOST = "DOZZLE_HOST";
export const sessionHost = useSessionStorage<string | null>(DOZZLE_HOST, null);

if (config.hosts.length === 1 && !sessionHost.value) {
  sessionHost.value = config.hosts[0].id;
}

export function persistentVisibleKeysForContainer(container: Ref<Container>): Ref<string[][]> {
  const storage = useProfileStorage("visibleKeys", {});
  return computed(() => {
    if (!(container.value.storageKey in storage.value)) {
      // Returning a temporary ref here to avoid writing an empty array to storage
      const visibleKeys = ref<string[][]>([]);
      watchOnce(visibleKeys, () => (storage.value[container.value.storageKey] = visibleKeys.value), { deep: true });
      return visibleKeys.value;
    }

    return storage.value[container.value.storageKey];
  });
}

export const pinnedContainers = useProfileStorage("pinned", new Set<string>());
