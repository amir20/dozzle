import { type LogEntry } from "@/types/LogEntry";
import {  VisibleLogEntry } from "@/types/VisibleLogEntry";
import type { ComputedRef, Ref } from "vue";

export function useVisibleFilter(visibleKeys: ComputedRef<Ref<string[][]>>) {
  function filteredPayload(messages: Ref<LogEntry[]>) {
    return computed(() => {
      return messages.value.map((d) => new VisibleLogEntry(d, visibleKeys.value));
    });
  }

  return { filteredPayload };
}
