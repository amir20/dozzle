import { computed, ComputedRef, Ref } from "vue";
import { LogEntry } from "./LogEntry";
import { flattenJSON, getDeep } from "@/utils";

export class VisibleLogEntry implements LogEntry {
  private readonly entry: LogEntry;
  filteredPayload: undefined | ComputedRef<Record<string, any>>;

  constructor(entry: LogEntry, visibleKeys: Ref<string[][]>) {
    this.entry = entry;
    this.filteredPayload = undefined;
    if (this.entry.payload) {
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

  public hasPayload(): this is { payload: Record<string, any> } {
    return this.entry.payload !== undefined;
  }

  public get unfilteredPayload(): Record<string, any> | undefined {
    return this.entry.payload;
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

  public get id(): string {
    return this.entry.id;
  }

  public get event(): string | undefined {
    return this.entry.event;
  }
}
