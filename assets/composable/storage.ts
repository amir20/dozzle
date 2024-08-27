import { Container } from "@/models/Container";

const DOZZLE_HOST = "DOZZLE_HOST";
export const sessionHost = useSessionStorage<string | null>(DOZZLE_HOST, null);

if (config.hosts.length === 1 && !sessionHost.value) {
  sessionHost.value = config.hosts[0].id;
}

const storage = useProfileStorage("visibleKeys", {});
export function persistentVisibleKeysForContainer(container: Ref<Container>): Ref<string[][]> {
  // Computed property to only store to storage when the value changes
  return computed({
    get: () => {
      return storage.value[container.value.storageKey] || [];
    },
    set: (value: string[][]) => {
      storage.value[container.value.storageKey] = value;
    },
  });
}

export const pinnedContainers = useProfileStorage("pinned", new Set<string>());
