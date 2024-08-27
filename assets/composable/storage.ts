import { Container } from "@/models/Container";

const DOZZLE_HOST = "DOZZLE_HOST";
export const sessionHost = useSessionStorage<string | null>(DOZZLE_HOST, null);

if (config.hosts.length === 1 && !sessionHost.value) {
  sessionHost.value = config.hosts[0].id;
}

const storage = useProfileStorage("visibleKeys", new Map<string, Map<string[], boolean>>(), {
  from(transformed: [string, [string[], boolean][]][]) {
    return new Map(transformed.map(([key, value]) => [key, new Map(value)]));
  },
  to(value: Map<string, Map<string[], boolean>>) {
    const outer = Array.from(value.entries());
    const inner = outer.map(([key, value]) => [key, Array.from(value.entries())]);
    return inner;
  },
});
export function persistentVisibleKeysForContainer(container: Ref<Container>): Ref<Map<string[], boolean>> {
  // Computed property to only store to storage when the value changes
  return computed({
    get: () => storage.value.get(container.value.storageKey) || new Map<string[], boolean>(),
    set: (value: Map<string[], boolean>) => storage.value.set(container.value.storageKey, value),
  });
}

export const pinnedContainers = useProfileStorage("pinned", new Set<string>());
