/**
 * The container ids the main view is currently streaming: one for a container page,
 * many for a merged one, none anywhere else. Shared by the sidebar, which marks the
 * rows feeding the stream, and the popup and command palette, which add and remove them.
 */
export const useStreamedContainers = () => {
  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();

  const ids = computed(() => {
    if (route.name === "/merged/[ids]") return String(route.params.ids).split(",");
    if (route.name === "/container/[id]") return [String(route.params.id)];
    return [];
  });

  const isStreaming = (id: string) => ids.value.includes(id);

  // A single container is just the container page, so there is nothing to mark.
  const isMerged = computed(() => ids.value.length > 1);

  // Nothing to merge into from a dashboard or a group view, and removing the last
  // id would leave a merged view with no containers in it.
  const canToggle = (id: string) => ids.value.length > (isStreaming(id) ? 1 : 0);

  // Says why the toggle cannot be used, when it cannot.
  const toggleTitle = (id: string) => {
    if (ids.value.length === 0) return t("tooltip.merge-stream-hint");
    return isStreaming(id) ? t("tooltip.unmerge-stream") : t("tooltip.merge-stream");
  };

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

  return { ids, isStreaming, isMerged, canToggle, toggleTitle, toggle };
};
