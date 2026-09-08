/**
 * The container ids the main view is currently streaming: one for a container page,
 * many for a merged one, none anywhere else. Shared by the sidebar, which marks the
 * rows feeding the stream, and the popup, which adds and removes them.
 */
export const useStreamedContainers = () => {
  const route = useRoute();
  const router = useRouter();

  const ids = computed(() => {
    if (route.name === "/merged/[ids]") return String(route.params.ids).split(",");
    if (route.name === "/container/[id]") return [String(route.params.id)];
    return [];
  });

  const isStreaming = (id: string) => ids.value.includes(id);

  // A single container is just the container page, so there is nothing to mark.
  const isMerged = computed(() => ids.value.length > 1);

  const toggle = (id: string) => {
    const next = isStreaming(id) ? ids.value.filter((i) => i !== id) : [...ids.value, id];

    // Would leave a merged view with nothing in it.
    if (next.length === 0) return;

    router.push(
      next.length === 1
        ? { name: "/container/[id]", params: { id: next[0] } }
        : { name: "/merged/[ids]", params: { ids: next.join(",") } },
    );
  };

  return { ids, isStreaming, isMerged, toggle };
};
