/**
 * @vitest-environment jsdom
 */
import { describe, expect, test, vi, beforeEach, afterEach } from "vitest";
import {
  mergeAlerts,
  fetchAlerts,
  attachEvents,
  mergeCloudEvents,
  type CloudAlert,
  type CloudEvent,
} from "./cloudAlerts";
import { AlertLogEntry, CloudEventLogEntry, SimpleLogEntry, type LogEntry, type LogMessage } from "@/models/LogEntry";

vi.mock("@/composable/cloudConfig", () => ({ useCloudConfig: () => ({ cloudConfig: { value: null } }) }));

const ms = (n: number) => new Date(n);
const ns = (n: number) => n * 1_000_000;

function log(id: number, at: number, containerID = "abc"): LogEntry<LogMessage> {
  return new SimpleLogEntry(`line ${id}`, containerID, id, ms(at), "info", "stdout", `line ${id}`);
}

function alert(overrides: Partial<CloudAlert> = {}): CloudAlert {
  return {
    alertId: "1",
    containerId: "abc",
    hostId: "h",
    ts: ns(100),
    headline: "Pool exhausted",
    level: "error",
    eventCount: 3,
    createdAt: ns(100),
    isOrigin: true,
    ...overrides,
  };
}

const shapeOf = (entries: LogEntry<LogMessage>[]) =>
  entries.map((e) => (e instanceof AlertLogEntry ? `alert:${e.alert.alertId}` : `log:${e.id}`));

describe("mergeAlerts", () => {
  test("anchors an alert immediately after its trigger line", () => {
    const logs = [log(10, 100), log(11, 200), log(12, 300)];
    const merged = mergeAlerts(logs, [alert({ logId: 11, ts: ns(200) })], new Set());

    expect(shapeOf(merged)).toEqual(["log:10", "log:11", "alert:1", "log:12"]);
  });

  // The whole reason for matching on id rather than sorting by time: an alert
  // shares a millisecond with the line that caused it, so ordering by
  // timestamp alone would place it either side depending on sort stability.
  test("lands after the trigger line even when timestamps are identical", () => {
    const logs = [log(10, 100), log(11, 100), log(12, 100)];
    const merged = mergeAlerts(logs, [alert({ logId: 11, ts: ns(100) })], new Set());

    expect(shapeOf(merged)).toEqual(["log:10", "log:11", "alert:1", "log:12"]);
  });

  test("falls back to timestamp position when the trigger line is not loaded", () => {
    const logs = [log(10, 100), log(11, 300)];
    // logId 99 is not in this run — e.g. the line is outside the window.
    const merged = mergeAlerts(logs, [alert({ logId: 99, ts: ns(200) })], new Set());

    expect(shapeOf(merged)).toEqual(["log:10", "alert:1", "log:11"]);
  });

  test("positions metric alerts, which carry no trigger line, by timestamp", () => {
    const logs = [log(10, 100), log(11, 300)];
    const merged = mergeAlerts(logs, [alert({ logId: undefined, ts: ns(200) })], new Set());

    expect(shapeOf(merged)).toEqual(["log:10", "alert:1", "log:11"]);
  });

  test("ignores a trigger line belonging to a different container", () => {
    const logs = [log(10, 100), log(11, 300, "other")];
    // Same FNV id on a different container must not steal the anchor.
    const merged = mergeAlerts(logs, [alert({ logId: 11, ts: ns(400) })], new Set());

    expect(shapeOf(merged)).toEqual(["log:10", "log:11", "alert:1"]);
  });

  test("appends alerts that fall past the end of the run", () => {
    const logs = [log(10, 100)];
    const merged = mergeAlerts(logs, [alert({ ts: ns(900) })], new Set());

    expect(shapeOf(merged)).toEqual(["log:10", "alert:1"]);
  });

  test("keeps time-anchored alerts in order among themselves", () => {
    const logs = [log(10, 500)];
    const merged = mergeAlerts(
      logs,
      [alert({ alertId: "2", ts: ns(300) }), alert({ alertId: "1", ts: ns(100) })],
      new Set(),
    );

    expect(shapeOf(merged)).toEqual(["alert:1", "alert:2", "log:10"]);
  });

  describe("dedupe", () => {
    test("does not place the same anchor twice across overlapping windows", () => {
      const seen = new Set<string>();
      const first = mergeAlerts([log(10, 100)], [alert({ logId: 10, ts: ns(100) })], seen);
      expect(shapeOf(first)).toEqual(["log:10", "alert:1"]);

      // Same alert returned again by an overlapping scroll window.
      const second = mergeAlerts([log(10, 100)], [alert({ logId: 10, ts: ns(100) })], seen);
      expect(shapeOf(second)).toEqual(["log:10"]);
    });

    // Keying the seen-set on alertId alone would swallow this: scrolling
    // newest -> oldest loads the follow-up first, and the origin — the thing
    // the user is actually scrolling back to find — would never render.
    test("still places the origin after a follow-up anchor of the same incident", () => {
      const seen = new Set<string>();
      const followUp = alert({ ts: ns(900), isOrigin: false });
      const origin = alert({ ts: ns(100), isOrigin: true });

      mergeAlerts([log(20, 900)], [followUp], seen);
      const older = mergeAlerts([log(10, 100)], [origin], seen);

      expect(shapeOf(older)).toEqual(["alert:1", "log:10"]);
      expect((older[0] as AlertLogEntry).alert.isOrigin).toBe(true);
    });
  });

  // A follow-up anchor marks an incident that was already open. The per-line
  // badges say that with more precision, and a block here would claim a
  // delivery that never happened.
  test("drops follow-up anchors, keeping only origins", () => {
    const logs = [log(10, 100)];
    const merged = mergeAlerts(logs, [alert({ isOrigin: false, ts: ns(100) })], new Set());
    expect(shapeOf(merged)).toEqual(["log:10"]);
  });

  test("still places the origin of the same incident", () => {
    const logs = [log(10, 100)];
    const merged = mergeAlerts(
      logs,
      [alert({ alertId: "1", isOrigin: false, ts: ns(50) }), alert({ alertId: "1", isOrigin: true, ts: ns(100) })],
      new Set(),
    );
    expect(shapeOf(merged)).toEqual(["alert:1", "log:10"]);
  });

  test("returns the original array when there is nothing to merge", () => {
    const logs = [log(10, 100)];
    expect(mergeAlerts(logs, [], new Set())).toBe(logs);
  });
});

describe("fetchAlerts", () => {
  const originalFetch = global.fetch;
  beforeEach(() => vi.stubGlobal("withBase", (s: string) => s));
  afterEach(() => {
    global.fetch = originalFetch;
    vi.unstubAllGlobals();
  });

  // An unlinked Dozzle must put no traffic on the network at all — not a
  // request that 503s, not one that returns empty. This is the backstop; the
  // callers in logLoader check availability before assembling a window.
  test("does not call cloud when not linked", async () => {
    const spy = vi.fn();
    global.fetch = spy;

    expect(await fetchAlerts(["abc"], ms(0), ms(1), { linked: false })).toEqual({ alerts: [], events: [] });
    expect(spy).not.toHaveBeenCalled();
  });

  test("does not call cloud with no containers", async () => {
    const spy = vi.fn();
    global.fetch = spy;

    expect(await fetchAlerts([], ms(0), ms(1), { linked: true })).toEqual({ alerts: [], events: [] });
    expect(spy).not.toHaveBeenCalled();
  });

  // This rides the scroll path, where the logs have already rendered. A cloud
  // outage must cost the user their alerts, never their logs.
  test("degrades to no alerts when cloud fails", async () => {
    global.fetch = vi.fn().mockRejectedValue(new Error("cloud down"));
    expect(await fetchAlerts(["abc"], ms(0), ms(1), { linked: true })).toEqual({ alerts: [], events: [] });

    global.fetch = vi.fn().mockResolvedValue({ ok: false, status: 502 });
    expect(await fetchAlerts(["abc"], ms(0), ms(1), { linked: true })).toEqual({ alerts: [], events: [] });
  });

  test("sends the window as unix nanoseconds", async () => {
    const spy = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ hits: [] }) });
    global.fetch = spy;

    await fetchAlerts(["abc", "def"], ms(1000), ms(2000), { linked: true });

    const url = spy.mock.calls[0][0] as string;
    expect(url).toContain("containerIds=abc%2Cdef");
    expect(url).toContain(`from=${ns(1000)}`);
    expect(url).toContain(`to=${ns(2000)}`);
  });
});

describe("attachEvents", () => {
  const event = (overrides: Partial<CloudEvent> = {}): CloudEvent => ({
    ts: ns(100),
    logId: 10,
    containerId: "abc",
    suppressed: true,
    ...overrides,
  });

  test("badges the line the event matched", () => {
    const logs = [log(10, 100), log(11, 200)];
    expect(attachEvents(logs, [event()])).toBe(true);
    expect(logs[0].matchedEvent).toEqual({ suppressed: true });
    expect(logs[1].matchedEvent).toBeUndefined();
  });

  // The FNV id is only unique within a container, so the same hash on another
  // container must not steal the badge.
  test("ignores a matching id on a different container", () => {
    const logs = [log(10, 100, "other")];
    expect(attachEvents(logs, [event()])).toBe(false);
    expect(logs[0].matchedEvent).toBeUndefined();
  });

  // Metric and event notifications have no line to badge; guessing one from a
  // timestamp would mark an unrelated log line.
  test("skips events with no log line", () => {
    const logs = [log(10, 100)];
    expect(attachEvents(logs, [event({ logId: undefined })])).toBe(false);
    expect(logs[0].matchedEvent).toBeUndefined();
  });

  // An event that produced an alert is already represented by the alert block;
  // badging its line too would say the same thing twice.
  test("does not badge a line whose event produced an alert", () => {
    const logs = [log(10, 100)];
    expect(attachEvents(logs, [event({ suppressed: false, alertId: "7" })])).toBe(false);
    expect(logs[0].matchedEvent).toBeUndefined();
  });

  // Metric and container events have no line of their own — they get rows.
  test("does not badge log lines with non-log events", () => {
    const logs = [log(10, 100)];
    expect(attachEvents(logs, [event({ type: "metric" })])).toBe(false);
    expect(logs[0].matchedEvent).toBeUndefined();
  });

  test("reports no change when there is nothing to badge", () => {
    expect(attachEvents([log(10, 100)], [])).toBe(false);
  });
});

describe("mergeCloudEvents", () => {
  const cloudEvent = (overrides: Partial<CloudEvent> = {}): CloudEvent => ({
    ts: ns(200),
    containerId: "abc",
    type: "metric",
    detail: "CPU: 91.4%, Memory: 62.0%",
    suppressed: true,
    ...overrides,
  });

  const shape = (entries: LogEntry<LogMessage>[]) =>
    entries.map((e) => (e instanceof CloudEventLogEntry ? `event:${e.event.type}` : `log:${e.id}`));

  // A metric notification has no log line to badge — it IS the event — so
  // unlike a log event it can only be shown as a row.
  test("splices a metric event in by timestamp", () => {
    const logs = [log(10, 100), log(11, 300)];
    expect(shape(mergeCloudEvents(logs, [cloudEvent()], new Set()))).toEqual(["log:10", "event:metric", "log:11"]);
  });

  test("splices a container event in the same way", () => {
    const logs = [log(10, 100)];
    const merged = mergeCloudEvents(logs, [cloudEvent({ type: "event", ts: ns(400) })], new Set());
    expect(shape(merged)).toEqual(["log:10", "event:event"]);
  });

  // Log events are badged on their own line; giving them a row too would
  // duplicate a line already on screen.
  test("leaves log events alone", () => {
    const logs = [log(10, 100)];
    expect(mergeCloudEvents(logs, [cloudEvent({ type: "log", logId: 10 })], new Set())).toBe(logs);
  });

  test("skips events that produced an alert", () => {
    const logs = [log(10, 100)];
    expect(mergeCloudEvents(logs, [cloudEvent({ suppressed: false })], new Set())).toBe(logs);
  });

  test("does not place the same event twice across overlapping windows", () => {
    const seen = new Set<string>();
    const logs = [log(10, 100)];
    expect(shape(mergeCloudEvents(logs, [cloudEvent()], seen))).toHaveLength(2);
    expect(mergeCloudEvents(logs, [cloudEvent()], seen)).toBe(logs);
  });
});
