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
export type Position = "start" | "end" | "middle" | undefined;
export interface LogEvent {
  readonly m: string | JSONObject;
  readonly ts: number;
  readonly id: number;
  readonly l: string;
  readonly p: Position;
}

export abstract class LogEntry<T extends string | JSONObject> implements HasComponent {
  protected readonly _message: T;
  constructor(message: T, public readonly id: number, public readonly date: Date, public readonly level?: string) {
    this._message = message;
  }

  public get message(): T {
    return this._message;
  }

  abstract getComponent(): Component;
}

export class SimpleLogEntry extends LogEntry<string> {
  constructor(
    message: string,
    id: number,
    date: Date,
    public readonly level: string,
    public readonly position: Position
  ) {
    super(message, id, date, level);
  }
  getComponent(): Component {
    return SimpleLogItem;
  }
}

export class ComplexLogEntry extends LogEntry<JSONObject> {
  private readonly filteredMessage: ComputedRef<JSONObject>;

  constructor(
    message: JSONObject,
    id: number,
    date: Date,
    public readonly level: string,
    visibleKeys?: Ref<string[][]>
  ) {
    super(message, id, date, level);
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
    return new ComplexLogEntry(event._message, event.id, event.date, event.level, visibleKeys);
  }
}

export class DockerEventLogEntry extends LogEntry<string> {
  constructor(message: string, date: Date, public readonly event: string) {
    super(message, date.getTime(), date, "info");
  }
  getComponent(): Component {
    return DockerEventLogItem;
  }
}

export class SkippedLogsEntry extends LogEntry<string> {
  private _totalSkipped = 0;
  private lastSkipped: LogEntry<string | JSONObject>;

  constructor(
    date: Date,
    totalSkipped: number,
    public readonly firstSkipped: LogEntry<string | JSONObject>,
    lastSkipped: LogEntry<string | JSONObject>
  ) {
    super("", date.getTime(), date, "info");
    this._totalSkipped = totalSkipped;
    this.lastSkipped = lastSkipped;
  }
  getComponent(): Component {
    return SkippedEntriesLogItem;
  }

  public get message(): string {
    return `Skipped ${this.totalSkipped} entries`;
  }

  public addSkippedEntries(totalSkipped: number, lastItem: LogEntry<string | JSONObject>) {
    this._totalSkipped += totalSkipped;
    this.lastSkipped = lastItem;
  }

  public get totalSkipped(): number {
    return this._totalSkipped;
  }
}

export function asLogEntry(event: LogEvent): LogEntry<string | JSONObject> {
  if (typeof event.m === "string") {
    return new SimpleLogEntry(event.m, event.id, new Date(event.ts), event.l, event.p);
  } else {
    return new ComplexLogEntry(event.m, event.id, new Date(event.ts), event.l);
  }
}
