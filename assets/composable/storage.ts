import { Container } from "@/models/Container";

const DOZZLE_HOST = "DOZZLE_HOST";
export const sessionHost = useSessionStorage<string | null>(DOZZLE_HOST, null);

if (config.hosts.length === 1 && !sessionHost.value) {
  sessionHost.value = config.hosts[0].id;
}

export function persistentVisibleKeys(container: Ref<Container>) {
  const storage = useStorage<{ [key: string]: string[][] }>("DOZZLE_VISIBLE_KEYS", {});
  return computed(() => {
    if (!(container.value.storageKey in storage.value)) {
      storage.value[container.value.storageKey] = [];
    }

    return storage.value[container.value.storageKey];
  });
}

export const pinnedContainers = usePersistedStorage("pinned", new Set<string>());
