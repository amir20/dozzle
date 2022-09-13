import { Component, ComputedRef, Ref } from "vue";
import { flattenJSON, getDeep } from "@/utils";
import ComplexLogItem from "@/components/LogViewer/ComplexLogItem.vue";
import SimpleLogItem from "@/components/LogViewer/SimpleLogItem.vue";
import DockerEventLogItem from "@/components/LogViewer/DockerEventLogItem.vue";
import SkippedEntriesLogItem from "@/components/LogViewer/SkippedEntriesLogItem.vue";

export interface HasComponent {
  getComponent(): Component;
}

export type JSONValue = string | number | boolean | JSONObject | Array<JSONValue>;
export type JSONObject = { [x: string]: JSONValue };

export interface LogEvent {
  readonly m: string | JSONObject;
  readonly ts: number;
  readonly id: number;
}

export abstract class LogEntry<T extends string | JSONObject> implements HasComponent {
  protected readonly _message: T;
  constructor(message: T, public readonly id: number, public readonly date: Date) {
    this._message = message;
  }

  public get message(): T {
    return this._message;
  }

  abstract getComponent(): Component;
}

export class SimpleLogEntry extends LogEntry<string> {
  getComponent(): Component {
    return SimpleLogItem;
  }
}

export class ComplexLogEntry extends LogEntry<JSONObject> {
  private readonly filteredMessage: ComputedRef<JSONObject>;

  constructor(message: JSONObject, id: number, date: Date, visibleKeys?: Ref<string[][]>) {
    super(message, id, date);
    if (visibleKeys) {
      this.filteredMessage = computed(() => {
        if (!visibleKeys.value.length) {
          return flattenJSON(message);
        } else {
          return visibleKeys.value.reduce((acc, attr) => ({ ...acc, [attr.join(".")]: getDeep(message, attr) }), {});
        }
      });
    } else {
      this.filteredMessage = computed(() => flattenJSON(message));
    }
  }
  getComponent(): Component {
    return ComplexLogItem;
  }

  public get message(): JSONObject {
    return this.filteredMessage.value;
  }

  public get unfilteredMessage(): JSONObject {
    return this._message;
  }

  static fromLogEvent(event: ComplexLogEntry, visibleKeys: Ref<string[][]>): ComplexLogEntry {
    return new ComplexLogEntry(event._message, event.id, event.date, visibleKeys);
  }
}

export class DockerEventLogEntry extends LogEntry<string> {
  constructor(message: string, date: Date, public readonly event: string) {
    super(message, date.getTime(), date);
  }
  getComponent(): Component {
    return DockerEventLogItem;
  }
}

export class SkippedLogsEntry extends LogEntry<string> {
  private totalSkipped = 0;
  private lastSkipped: LogEntry<string | JSONObject>;

  constructor(
    date: Date,
    totalSkipped: number,
    public readonly firstSkipped: LogEntry<string | JSONObject>,
    lastSkipped: LogEntry<string | JSONObject>
  ) {
    super("", date.getTime(), date);
    this.totalSkipped = totalSkipped;
    this.lastSkipped = lastSkipped;
  }
  getComponent(): Component {
    return SkippedEntriesLogItem;
  }

  public get message(): string {
    return `Skipped ${this.totalSkipped} entries`;
  }

  public addSkippedEntries(totalSkipped: number, lastItem: LogEntry<string | JSONObject>) {
    this.totalSkipped += totalSkipped;
    this.lastSkipped = lastItem;
  }
}

export function asLogEntry(event: LogEvent): LogEntry<string | JSONObject> {
  if (typeof event.m === "string") {
    return new SimpleLogEntry(event.m, event.id, new Date(event.ts));
  } else {
    return new ComplexLogEntry(event.m, event.id, new Date(event.ts));
  }
}
