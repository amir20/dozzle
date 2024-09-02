import { ComplexLogEntry, type JSONObject, type LogEntry } from "@/models/LogEntry";

export function useVisibleFilter(visibleKeys: Ref<Map<string[], boolean>>) {
  function filteredPayload(messages: Ref<LogEntry<string | JSONObject>[]>) {
    return computed(() => {
      return messages.value.map((d) => {
        if (d instanceof ComplexLogEntry) {
          return ComplexLogEntry.fromLogEvent(d, visibleKeys);
        } else {
          return d;
        }
      });
    });
  }

  return { filteredPayload };
}

// TODO clean up search filter to have complex items also be filtered with visible keys
