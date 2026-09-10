import { ComplexLogEntry, type LogMessage, type LogEntry } from "@/models/LogEntry";

export type VisibleKeysSource = Map<string[], boolean> | ((containerID: string) => Map<string[], boolean>);

/**
 * Visible keys for multi-container viewers. Each log line resolves its own container so
 * field toggles made on a single container view carry over here.
 */
export function useVisibleKeysByContainer(): (containerID: string) => Map<string[], boolean> {
  const { findContainerById } = useContainerStore();
  return (containerID: string) => visibleKeysForContainer(findContainerById(containerID));
}

export function useVisibleFilter(visibleKeys: Ref<VisibleKeysSource>) {
  const { isSearching, inverseFilter } = useSearchFilter();
  function filteredPayload(messages: Ref<LogEntry<LogMessage>[]>) {
    return computed(() => {
      return messages.value
        .map((d) => {
          if (d instanceof ComplexLogEntry) {
            const keys = toRef(() => {
              const source = visibleKeys.value;
              return typeof source === "function" ? source(d.containerID) : source;
            });
            return ComplexLogEntry.fromLogEvent(d, keys);
          } else {
            return d;
          }
        })
        .filter((d) => {
          if (isSearching.value && d instanceof ComplexLogEntry) {
            const hasMark = Object.values(d.message).some((v) => JSON.stringify(v)?.includes("<mark>"));
            return inverseFilter.value ? !hasMark : hasMark;
          } else {
            return true;
          }
        });
    });
  }

  return { filteredPayload };
}
