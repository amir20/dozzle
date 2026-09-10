/**
 * True when the viewer shares its row with another column: a pinned log, or the
 * chat pane.
 *
 * A column that shares the row has to own its scrolling. Otherwise the document
 * scrolls and every column scrolls with it, so the chat pane's header leaves
 * the top of the screen and its composer is stranded halfway down a page whose
 * logs are somewhere else entirely.
 */
export function useSplitColumns() {
  const { pinnedLogs } = storeToRefs(usePinnedLogsStore());
  const { open: chatOpen } = useCloudChat();

  return computed(() => pinnedLogs.value.length > 0 || (chatOpen.value && !isMobile.value));
}
