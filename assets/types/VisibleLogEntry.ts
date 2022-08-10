import { computed, Ref } from "vue";
import { LogEntry } from "./LogEntry";
import { flattenJSON, getDeep } from "@/utils";

export class VisibleLogEntry {
  entry: LogEntry;
  filteredPayload: undefined | Ref<Record<string, any>>;

  constructor(entry: LogEntry, visibleKeys: Ref<string[][]>) {
    this.entry = entry;
    this.filteredPayload = undefined;
    if (this.entry.payload !== undefined) {
      this.filteredPayload = computed(() => {
        if (!visibleKeys.value.length) {
          return flattenJSON(this.entry.payload!);
        } else {
          return visibleKeys.value.reduce(
            (acc, attr) => ({ ...acc, [attr.join(".")]: getDeep(this.entry.payload!, attr) }),
            {}
          );
        }
      });
    }
  }

  public get isJSON(): boolean {
    return this.entry.payload !== undefined;
  }
}
