import { Container } from "@/models/Container";

export const sessionHost = useSessionStorage<string | null>("host", null);

if (config.hosts.length === 1 && !sessionHost.value) {
  sessionHost.value = config.hosts[0].id;
}

export function persistentVisibleKeys(container: ComputedRef<Container>) {
  return computed(() => useStorage(container.value.storageKey, []));
}

const DOZZLE_PINNED_CONTAINERS = "DOZZLE_PINNED_CONTAINERS";
export const pinnedContainers = useStorage(DOZZLE_PINNED_CONTAINERS, new Set<string>());

export function togglePinnedContainer(id: string) {
  if (pinnedContainers.value.has(id)) {
    pinnedContainers.value.delete(id);
  } else {
    pinnedContainers.value.add(id);
  }
}
