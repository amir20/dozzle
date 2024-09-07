import { ComplexLogEntry, type JSONObject, type LogEntry } from "@/models/LogEntry";

export function useVisibleFilter(visibleKeys: Ref<Map<string[], boolean>>) {
  const { isSearching } = useSearchFilter();
  function filteredPayload(messages: Ref<LogEntry<string | JSONObject>[]>) {
    return computed(() => {
      return messages.value
        .map((d) => {
          if (d instanceof ComplexLogEntry) {
            return ComplexLogEntry.fromLogEvent(d, visibleKeys);
          } else {
            return d;
          }
        })
        .filter((d) => {
          if (isSearching.value && d instanceof ComplexLogEntry) {
            return Object.values(d.message).some((v) => JSON.stringify(v)?.includes("<mark>"));
          } else {
            return true;
          }
        });
    });
  }

  return { filteredPayload };
}
