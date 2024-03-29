import { ComplexLogEntry, type JSONObject, type LogEntry } from "@/models/LogEntry";
import type { Ref } from "vue";

export function useVisibleFilter(visibleKeys: Ref<string[][]>) {
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
