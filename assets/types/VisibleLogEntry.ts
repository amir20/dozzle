import { computed, ComputedRef, Ref } from "vue";
import { flattenJSON, getDeep } from "@/utils";
import type { JSONObject, LogEntry } from "./LogEntry";

export class VisibleLogEntry implements LogEntry {
  private readonly entry: LogEntry;
  filteredMessage: undefined | ComputedRef<Record<string, any>>;

  constructor(entry: LogEntry, visibleKeys: Ref<string[][]>) {
    this.entry = entry;
    this.filteredMessage = undefined;
    if (this.isComplex()) {
      const message = this.message;
      this.filteredMessage = computed(() => {
        if (!visibleKeys.value.length) {
          return flattenJSON(message);
        } else {
          return visibleKeys.value.reduce((acc, attr) => ({ ...acc, [attr.join(".")]: getDeep(message, attr) }), {});
        }
      });
    }
  }

  public isComplex(): this is { message: JSONObject } {
    return typeof this.entry.message === "object";
  }

  public isSimple(): this is { message: string } {
    return !this.isComplex();
  }

  public get unfilteredPayload(): JSONObject {
    if (typeof this.entry.message === "string") {
      throw new Error("Cannot get unfiltered payload of a simple message");
    }
    return this.entry.message;
  }

  public get date(): Date {
    return this.entry.date;
  }

  public get message(): string | JSONObject {
    return this.filteredMessage?.value ?? this.entry.message;
  }

  public get id(): number {
    return this.entry.id;
  }

  public get event(): string | undefined {
    return this.entry.event;
  }
}
