import { computed, ComputedRef, Ref } from "vue";
import { LogEntry } from "./LogEntry";
import { flattenJSON, getDeep } from "@/utils";

export class VisibleLogEntry implements LogEntry {
  readonly entry: LogEntry;
  filteredPayload: undefined | ComputedRef<Record<string, any>>;

  constructor(entry: LogEntry, visibleKeys: Ref<string[][]>) {
    this.entry = entry;
    this.filteredPayload = undefined;
    if (this.hasPayload()) {
      const payload = this.entry.payload;
      this.filteredPayload = computed(() => {
        if (!visibleKeys.value.length) {
          return flattenJSON(payload);
        } else {
          return visibleKeys.value.reduce((acc, attr) => ({ ...acc, [attr.join(".")]: getDeep(payload, attr) }), {});
        }
      });
    }
  }

  public hasPayload(): this is { entry: { payload: Record<string, any> } } {
    return this.entry.payload !== undefined;
  }

  public get payload(): Record<string, any> | undefined {
    return this.filteredPayload?.value;
  }

  public get date(): Date {
    return this.entry.date;
  }

  public get message(): string | undefined {
    return this.entry.message;
  }

  public get key(): string {
    return this.entry.key;
  }
}
