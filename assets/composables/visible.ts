import { ref, computed, Ref } from "vue";
import { LogEntry } from "@/types/LogEntry";
import { VisibleLogEntry } from "@/types/VisibleLogEntry";

export function useVisibleFilter(visibleKeys: Ref<string[][]>) {
  function filteredPayload(messages: Ref<LogEntry[]>) {
    return computed(() => {
      return messages.value.map((d) => new VisibleLogEntry(d, visibleKeys));
    });
  }

  return { filteredPayload };
}
