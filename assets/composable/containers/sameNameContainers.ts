import { Container } from "@/models/Container";

/**
 * Every container Docker still has under this one's name and custom group, the given one
 * included, newest first. Usually the same service restarted or recreated.
 */
export function useSameNameContainers(container: MaybeRefOrGetter<Container>) {
  const store = useContainerStore();
  const { containers } = storeToRefs(store);
  return computed(() => {
    const { name, customGroup } = toValue(container);
    return containers.value
      .filter((c) => c.name === name && c.customGroup === customGroup)
      .sort((a, b) => +b.created - +a.created);
  });
}
