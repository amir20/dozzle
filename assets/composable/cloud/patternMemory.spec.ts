/**
 * @vitest-environment jsdom
 */
import { describe, expect, test, vi, afterEach } from "vitest";
import { ref } from "vue";
import {
  attachPatternMemory,
  fetchPatternContext,
  linesNeedingMemory,
  chipMoment,
  MAX_PATTERN_LINES,
  type PatternContextHit,
} from "./patternMemory";
import {
  AlertLogEntry,
  ComplexLogEntry,
  SimpleLogEntry,
  type Level,
  type LogEntry,
  type LogMessage,
} from "@/models/LogEntry";

function log(id: number, level: Level = "error", containerID = "abc"): LogEntry<LogMessage> {
  return new SimpleLogEntry(`line ${id}`, containerID, id, new Date(id), level, "stderr", `line ${id}`);
}

function hit(overrides: Partial<PatternContextHit> = {}): PatternContextHit {
  return {
    containerId: "abc",
    logIds: [1],
    pattern: "connection refused to <IP4>:<N>",
    level: "error",
    status: "new",
    ratePerHour: 4,
    usualRatePerHour: 0,
    ...overrides,
  };
}

afterEach(() => vi.unstubAllGlobals());

describe("linesNeedingMemory", () => {
  test("asks only about error and warn lines, newest first", () => {
    const logs = [log(1, "error"), log(2, "info"), log(3, "warn"), log(4, "debug"), log(5, "fatal")];
    expect(linesNeedingMemory(logs).map((l) => l.id)).toEqual([5, 3, 1]);
  });

  test("never asks about an alert row, whose id is a timestamp", () => {
    const ts = 1_791_146_775_319_000_000;
    const alert = new AlertLogEntry(
      {
        alertId: "a1",
        containerId: "abc",
        hostId: "h",
        ts,
        headline: "db down",
        level: "error",
        eventCount: 1,
        createdAt: ts,
        isOrigin: true,
      },
      new Date(ts / 1_000_000),
    );
    expect(linesNeedingMemory([log(1), alert]).map((l) => l.id)).toEqual([1]);
  });

  test("skips lines that already have an answer", () => {
    const a = log(1);
    a.patternMemory = { status: "known", pattern: "x", ratePerHour: 0, usualRatePerHour: 0 };
    expect(linesNeedingMemory([a, log(2)]).map((l) => l.id)).toEqual([2]);
  });

  test("caps a request at the server's limit", () => {
    const logs = Array.from({ length: MAX_PATTERN_LINES + 20 }, (_, i) => log(i + 1));
    const asked = linesNeedingMemory(logs);
    expect(asked).toHaveLength(MAX_PATTERN_LINES);
    expect(asked[0].id).toBe(MAX_PATTERN_LINES + 20);
  });

  test("stops asking about a line Cloud never answers", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(JSON.stringify({ patterns: [] }), { status: 200 })),
    );
    const line = log(1);
    for (let i = 0; i < 3; i++) await fetchPatternContext(linesNeedingMemory([line]), new Date(0), new Date(1));
    expect(linesNeedingMemory([line])).toEqual([]);
  });
});

describe("fetchPatternContext", () => {
  test("posts container and log ids with a nanosecond window", async () => {
    const fetchMock = vi.fn(async () => new Response(JSON.stringify({ patterns: [hit()] }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const got = await fetchPatternContext([log(7)], new Date(1000), new Date(2000));
    expect(got).toHaveLength(1);
    const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
    expect(JSON.parse(init.body as string)).toEqual({
      lines: [{ containerId: "abc", logId: 7 }],
      from: 1000 * 1_000_000,
      to: 2000 * 1_000_000,
    });
  });

  test("degrades to nothing on 204, errors and network failures", async () => {
    for (const impl of [
      async () => new Response(null, { status: 204 }),
      async () => new Response("boom", { status: 502 }),
      async () => {
        throw new Error("offline");
      },
    ]) {
      vi.stubGlobal("fetch", vi.fn(impl));
      expect(await fetchPatternContext([log(1)], new Date(0), new Date(1))).toEqual([]);
    }
  });
});

describe("attachPatternMemory", () => {
  test("matches on container and log id", () => {
    const mine = log(1, "error", "abc");
    const otherContainer = log(1, "error", "def");
    const changed = attachPatternMemory([mine, otherContainer], [hit({ logIds: [1] })]);
    expect(changed).toBe(true);
    expect(mine.patternMemory?.status).toBe("new");
    expect(otherContainer.patternMemory).toBeUndefined();
  });

  test("reports no change when nothing matched", () => {
    expect(attachPatternMemory([log(1)], [hit({ logIds: [99] })])).toBe(false);
    expect(attachPatternMemory([log(1)], [])).toBe(false);
  });

  // The viewer re-clones complex entries on every render; memory attached to
  // the original has to reach the clone, or a JSON line never shows its chip.
  test("survives the complex-entry clone", () => {
    const original = new ComplexLogEntry({ msg: "boom" }, "abc", 1, new Date(1), "error", "stderr", "{}", undefined);
    attachPatternMemory([original], [hit()]);
    const clone = ComplexLogEntry.fromLogEvent(original, ref(new Map()));
    expect(clone.patternMemory?.status).toBe("new");
  });
});

describe("chipMoment", () => {
  const memory = (firstSeenMs?: number) => ({
    status: "new" as const,
    pattern: "x",
    ratePerHour: 0,
    usualRatePerHour: 0,
    firstSeen: firstSeenMs === undefined ? undefined : firstSeenMs * 1_000_000,
  });

  test("lands on the first occurrence when it is earlier than the line", () => {
    const line = log(10_000_000);
    expect(chipMoment(memory(4_000_000), line)).toEqual({ containerId: "abc", date: new Date(4_000_000) });
  });

  test("lands on the line itself when it is the first occurrence", () => {
    const line = log(10_000_000);
    expect(chipMoment(memory(10_000_000), line)).toEqual({ containerId: "abc", date: line.date, logId: line.id });
  });

  test("lands on the line itself without a first sighting", () => {
    const line = log(10_000_000);
    expect(chipMoment(memory(), line)).toEqual({ containerId: "abc", date: line.date, logId: line.id });
  });
});
