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
export function visibleKeysForContainer(container: Container | undefined): Map<string[], boolean> {
  return (container && storage.value.get(container.storageKey)) || new Map<string[], boolean>();
}

export function persistentVisibleKeysForContainer(container: Ref<Container | undefined>): Ref<Map<string[], boolean>> {
  // Computed property to only store to storage when the value changes
  return computed({
    get: () => visibleKeysForContainer(container.value),
    // The container can go away while its log lines are still on screen, e.g. a
    // replica is destroyed and the stream reconnects. Toggling a field then has
    // nowhere to persist to, so drop it instead of blowing up the drawer.
    set: (value: Map<string[], boolean>) => {
      if (container.value) {
        storage.value.set(container.value.storageKey, value);
      }
    },
  });
}

export const pinnedContainers = useProfileStorage("pinned", new Set<string>());

// Keyed by "image@remoteDigest" so dismissing an available update stays
// dismissed only until the image actually moves again.
export const dismissedImageUpdates = useProfileStorage("dismissedImageUpdates", new Set<string>());

// One-time nudge toward `dev.dozzle.url`, shown on containers that publish a port but
// carry no label. Dismissing is global — once you know about the label, you know.
export const dismissedLinkHint = useProfileStorage("dismissedLinkHint", false);
