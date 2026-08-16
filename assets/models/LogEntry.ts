import { Component, ComputedRef, Ref, ShallowRef } from "vue";
import type { CloudAlert } from "@/composable/cloudAlerts";
import { flattenJSON } from "@/utils";
import ComplexLogItem from "@/components/LogViewer/ComplexLogItem.vue";
import SimpleLogItem from "@/components/LogViewer/SimpleLogItem.vue";
import GroupedLogItem from "@/components/LogViewer/GroupedLogItem.vue";
import ContainerEventLogItem from "@/components/LogViewer/ContainerEventLogItem.vue";
import SkippedEntriesLogItem from "@/components/LogViewer/SkippedEntriesLogItem.vue";
import LoadMoreLogItem from "@/components/LogViewer/LoadMoreLogItem.vue";
import AlertLogItem from "@/components/LogViewer/AlertLogItem.vue";

export type JSONValue = string | number | boolean | JSONObject | Array<JSONValue>;
export type JSONObject = { [x: string]: JSONValue };
export type Std = "stdout" | "stderr";
export type LogType = "single" | "group" | "complex";
export type Position = "start" | "end" | "middle" | undefined;
export type LogMessage = string | string[] | JSONObject;
export type Level =
  "error" | "warn" | "warning" | "info" | "debug" | "trace" | "severe" | "critical" | "fatal" | "unknown";

export interface LogFragment {
  readonly m: string;
}

export interface LogEvent {
  readonly t: LogType;
  readonly m: string | LogFragment[] | JSONObject;
  readonly ts: number;
  readonly id: number;
  readonly l: Level;
  readonly s: "stdout" | "stderr" | "unknown";
  readonly c: string;
  readonly rm: string;
}

/**
 * The matched event behind a log line, when Cloud reports one. Attached to the
 * entry rather than rendered as its own row: the event IS this line, so a
 * separate marker would duplicate what is already on screen — and during a
 * storm would put one between every pair of lines.
 */
export interface MatchedEvent {
  alertId: number;
  suppressed: boolean;
  level: string;
}

/**
 * Matched events live beside the entries rather than on them.
 *
 * They have to be ref-backed at all because entries are plain class instances
 * inside a shallowRef array — assigning a plain field mutates an object Vue
 * is not tracking, and the row never re-renders. Keeping the ref out here
 * rather than in a field leaves the entry's own shape untouched, which matters
 * because these objects get structurally snapshotted in tests.
 */
const matchedEvents = new WeakMap<LogEntry<LogMessage>, ShallowRef<MatchedEvent | undefined>>();

export abstract class LogEntry<T extends LogMessage> {
  protected readonly _message: T;

  /** The event Cloud matched on this line, once the alert loader reports it. */
  public get matchedEvent(): MatchedEvent | undefined {
    return matchedEventRef(this).value;
  }
  public set matchedEvent(event: MatchedEvent | undefined) {
    matchedEventRef(this).value = event;
  }
  constructor(
    message: T,
    public readonly containerID: string,
    public readonly id: number,
    public readonly date: Date,
    public readonly std: Std,
    public readonly rawMessage: string,
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
    public readonly std: Std,
    public readonly rawMessage: string,
  ) {
    super(message, containerID, id, date, std, rawMessage, level);
  }
  getComponent(): Component {
    return SimpleLogItem;
  }
}

export class GroupedLogEntry extends LogEntry<string[]> {
  constructor(
    messages: string[],
    containerID: string,
    id: number,
    date: Date,
    public readonly level: Level,
    public readonly std: Std,
  ) {
    super(messages as any, containerID, id, date, std, "", level);
  }

  public get message(): string[] {
    return this._message as unknown as string[];
  }

  getComponent(): Component {
    return GroupedLogItem;
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
    public readonly rawMessage: string,
    visibleKeys?: Ref<Map<string[], boolean>>,
  ) {
    super(message, containerID, id, date, std, rawMessage, level);
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
      event.rawMessage,
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

/**
 * An alert from Dozzle Cloud, merged into the log stream at the point it fired.
 *
 * Cloud stores the FNV-32a hash of the line that triggered each alert — the
 * same id LogEvent.Id carries — so where that line is on screen the alert can
 * be spliced in directly after it rather than positioned by timestamp. Metric
 * and event alerts have no triggering line and carry logId 0, falling back to
 * their timestamp.
 *
 * An alert is an incident, not a single moment: Cloud folds follow-up batches
 * into the original alert, so one incident can have activity across many
 * windows. `isOrigin` distinguishes where it first fired from where it was
 * merely still firing.
 */
export class AlertLogEntry extends LogEntry<string> {
  constructor(
    public readonly alert: CloudAlert,
    date: Date,
  ) {
    // std/level feed the shared LogItem chrome. Alerts are not stream output,
    // but they are unmistakably not stdout either, and the level is the one
    // triage assigned.
    super(alert.headline, alert.containerId, alert.alertId, date, "stderr", alert.headline, alertLevel(alert.level));
  }

  getComponent(): Component {
    return AlertLogItem;
  }

  /**
   * Stable identity for dedupe across overlapping scroll windows. Keyed on the
   * ANCHOR as well as the alert: one incident legitimately marks every window
   * it was active in, and keying on alertId alone would let a follow-up anchor
   * loaded first swallow the origin loaded later.
   */
  public get anchorKey(): string {
    return `${this.alert.alertId}:${this.alert.ts}`;
  }
}

function alertLevel(level: string): Level {
  const known: Level[] = ["error", "warn", "warning", "info", "debug", "trace", "severe", "critical", "fatal"];
  return (known.find((l) => l === level) ?? "unknown") as Level;
}

export class SkippedLogsEntry extends LogEntry<string> {
  private _totalSkipped = ref(0);
  private lastSkipped: LogEntry<string | JSONObject>;

  constructor(
    date: Date,
    totalSkipped: number,
    public readonly firstSkipped: LogEntry<string | JSONObject>,
    lastSkipped: LogEntry<string | JSONObject>,
    private readonly loader: (i: SkippedLogsEntry) => Promise<void>,
  ) {
    super("", "", date.getTime(), date, "stderr", "info");
    this._totalSkipped.value = totalSkipped;
    this.lastSkipped = lastSkipped;
  }
  getComponent(): Component {
    return SkippedEntriesLogItem;
  }

  public get message(): string {
    return `Skipped ${this._totalSkipped.value} entries`;
  }

  public addSkippedEntries(totalSkipped: number, lastItem: LogEntry<string | JSONObject>) {
    this._totalSkipped.value += totalSkipped;
    this.lastSkipped = lastItem;
  }

  public get lastSkippedLog(): LogEntry<string | JSONObject> {
    return this.lastSkipped;
  }

  public async loadSkippedEntries(): Promise<void> {
    await this.loader(this);
  }

  public get totalSkipped(): number {
    return unref(this._totalSkipped);
  }
}

export class LoadMoreLogEntry extends LogEntry<string> {
  constructor(
    date: Date,
    private readonly loader: (i: LoadMoreLogEntry) => Promise<void>,
    public readonly rememberScrollPosition: boolean = true,
  ) {
    super("", "", date.getTime(), date, "stderr", "info");
  }

  getComponent(): Component {
    return LoadMoreLogItem;
  }

  async loadMore(): Promise<void> {
    await this.loader(this);
  }
}

function matchedEventRef(entry: LogEntry<any>): ShallowRef<MatchedEvent | undefined> {
  let existing = matchedEvents.get(entry);
  if (!existing) {
    existing = shallowRef<MatchedEvent | undefined>(undefined);
    matchedEvents.set(entry, existing);
  }
  return existing;
}

export function asLogEntry(event: LogEvent): LogEntry<LogMessage> {
  const std = event.s === "unknown" ? "stderr" : (event.s ?? "stderr");

  switch (event.t) {
    case "complex":
      return new ComplexLogEntry(event.m as JSONObject, event.c, event.id, new Date(event.ts), event.l, std, event.rm);
    case "group":
      return new GroupedLogEntry(
        (event.m as LogFragment[]).map((f) => f.m),
        event.c,
        event.id,
        new Date(event.ts),
        event.l,
        std,
      );
    case "single":
    default:
      return new SimpleLogEntry(event.m as string, event.c, event.id, new Date(event.ts), event.l, std, event.rm);
  }
}
