import { ComplexLogEntry, type LogMessage, type LogEntry } from "@/models/LogEntry";

export type VisibleKeysSource = Map<string[], boolean> | ((containerID: string) => Map<string[], boolean>);

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
