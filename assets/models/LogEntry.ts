import { Component, ComputedRef, Ref } from "vue";
import { flattenJSON } from "@/utils";
import ComplexLogItem from "@/components/LogViewer/ComplexLogItem.vue";
import SimpleLogItem from "@/components/LogViewer/SimpleLogItem.vue";
import ContainerEventLogItem from "@/components/LogViewer/ContainerEventLogItem.vue";
import SkippedEntriesLogItem from "@/components/LogViewer/SkippedEntriesLogItem.vue";

export type JSONValue = string | number | boolean | JSONObject | Array<JSONValue>;
export type JSONObject = { [x: string]: JSONValue };
export type Position = "start" | "end" | "middle" | undefined;
export type Std = "stdout" | "stderr";
export type Level =
  | "error"
  | "warn"
  | "warning"
  | "info"
  | "debug"
  | "trace"
  | "severe"
  | "critical"
  | "fatal"
  | "unknown";
export interface LogEvent {
  readonly m: string | JSONObject;
  readonly ts: number;
  readonly id: number;
  readonly l: Level;
  readonly p: Position;
  readonly s: "stdout" | "stderr" | "unknown";
  readonly c: string;
}

export abstract class LogEntry<T extends string | JSONObject> {
  protected readonly _message: T;
  constructor(
    message: T,
    public readonly containerID: string,
    public readonly id: number,
    public readonly date: Date,
    public readonly std: Std,
    public readonly level?: Level,
  ) {
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
    containerID: string,
    id: number,
    date: Date,
    public readonly level: Level,
    public readonly position: Position,
    public readonly std: Std,
  ) {
    super(message, containerID, id, date, std, level);
  }
  getComponent(): Component {
    return SimpleLogItem;
  }
}

export class ComplexLogEntry extends LogEntry<JSONObject> {
  private readonly filteredMessage: ComputedRef<Record<string, any>>;

  constructor(
    message: JSONObject,
    containerID: string,
    id: number,
    date: Date,
    public readonly level: Level,
    public readonly std: Std,
    visibleKeys?: Ref<Map<string[], boolean>>,
  ) {
    super(message, containerID, id, date, std, level);
    if (visibleKeys) {
      this.filteredMessage = computed(() => {
        if (visibleKeys.value.size === 0) {
          return flattenJSON(message);
        } else {
          const flatJSON = flattenJSON(message);
          const filteredJSON: Record<string, any> = {};
          for (const [keys, enabled] of visibleKeys.value.entries()) {
            const key = keys.join(".");
            if (!enabled) {
              delete flatJSON[key];
              continue;
            }
            filteredJSON[key] = flatJSON[key];
            delete flatJSON[key];
          }
          return { ...filteredJSON, ...flatJSON };
        }
      });
    } else {
      this.filteredMessage = computed(() => flattenJSON(message));
    }
  }
  getComponent(): Component {
    return ComplexLogItem;
  }

  public get message(): Record<string, any> {
    return unref(this.filteredMessage);
  }

  public get unfilteredMessage(): JSONObject {
    return this._message;
  }

  static fromLogEvent(event: ComplexLogEntry, visibleKeys: Ref<Map<string[], boolean>>): ComplexLogEntry {
    return new ComplexLogEntry(
      event._message,
      event.containerID,
      event.id,
      event.date,
      event.level,
      event.std,
      visibleKeys,
    );
  }
}

export class ContainerEventLogEntry extends LogEntry<string> {
  constructor(
    message: string,
    containerID: string,
    date: Date,
    public readonly event: "container-stopped" | "container-started",
  ) {
    super(message, containerID, date.getTime(), date, "stderr", "unknown");
  }
  getComponent(): Component {
    return ContainerEventLogItem;
  }
}

export class SkippedLogsEntry extends LogEntry<string> {
  private _totalSkipped = 0;
  private lastSkipped: LogEntry<string | JSONObject>;

  constructor(
    date: Date,
    totalSkipped: number,
    public readonly firstSkipped: LogEntry<string | JSONObject>,
    lastSkipped: LogEntry<string | JSONObject>,
  ) {
    super("", "", date.getTime(), date, "stderr", "info");
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

  public get lastSkippedItem(): LogEntry<string | JSONObject> {
    return this.lastSkipped;
  }
}

export function asLogEntry(event: LogEvent): LogEntry<string | JSONObject> {
  if (isObject(event.m)) {
    return new ComplexLogEntry(
      event.m,
      event.c,
      event.id,
      new Date(event.ts),
      event.l,
      event.s === "unknown" ? "stderr" : (event.s ?? "stderr"),
    );
  } else {
    return new SimpleLogEntry(
      event.m,
      event.c,
      event.id,
      new Date(event.ts),
      event.l,
      event.p,
      event.s === "unknown" ? "stderr" : (event.s ?? "stderr"),
    );
  }
}
