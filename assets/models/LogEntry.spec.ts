/**
 * @vitest-environment jsdom
 */
import { describe, expect, test } from "vitest";
import { ref } from "vue";
import {
  asLogEntry,
  ComplexLogEntry,
  GroupedLogEntry,
  SimpleLogEntry,
  SkippedLogsEntry,
  type LogEvent,
} from "./LogEntry";

function event(overrides: Partial<LogEvent> = {}): LogEvent {
  return {
    t: "single",
    m: "hello",
    ts: 1_700_000_000_000,
    id: 1,
    l: "info",
    s: "stdout",
    c: "container-1",
    rm: "raw",
    ...overrides,
  } as LogEvent;
}

describe("asLogEntry dispatch", () => {
  test("single -> SimpleLogEntry", () => {
    const entry = asLogEntry(event({ t: "single", m: "a line" }));
    expect(entry).toBeInstanceOf(SimpleLogEntry);
    expect(entry.message).toBe("a line");
  });

  test("group -> GroupedLogEntry with fragments mapped to strings", () => {
    const entry = asLogEntry(event({ t: "group", m: [{ m: "line1" }, { m: "line2" }] }));
    expect(entry).toBeInstanceOf(GroupedLogEntry);
    expect(entry.message).toEqual(["line1", "line2"]);
  });

  test("complex -> ComplexLogEntry", () => {
    const entry = asLogEntry(event({ t: "complex", m: { a: 1 } }));
    expect(entry).toBeInstanceOf(ComplexLogEntry);
  });

  test("unknown type falls back to SimpleLogEntry", () => {
    const entry = asLogEntry(event({ t: "mystery" as any, m: "x" }));
    expect(entry).toBeInstanceOf(SimpleLogEntry);
  });

  test("carries id, container, level and date from the event", () => {
    const entry = asLogEntry(event({ ts: 1_700_000_000_000, id: 7, c: "abc", l: "warn" }));
    expect(entry.id).toBe(7);
    expect(entry.containerID).toBe("abc");
    expect(entry.level).toBe("warn");
    expect(entry.date.getTime()).toBe(1_700_000_000_000);
  });
});

describe("std normalization", () => {
  test("unknown becomes stderr", () => {
    expect(asLogEntry(event({ s: "unknown" })).std).toBe("stderr");
  });

  test("missing becomes stderr", () => {
    expect(asLogEntry(event({ s: undefined as any })).std).toBe("stderr");
  });

  test("stdout and stderr are preserved", () => {
    expect(asLogEntry(event({ s: "stdout" })).std).toBe("stdout");
    expect(asLogEntry(event({ s: "stderr" })).std).toBe("stderr");
  });
});

describe("ComplexLogEntry filtering", () => {
  const message = { a: { b: 1 }, c: 2 };

  test("empty visibleKeys returns the fully flattened object", () => {
    const entry = new ComplexLogEntry(message, "c", 1, new Date(), "info", "stdout", "raw", ref(new Map()));
    expect(entry.message).toEqual({ "a.b": 1, c: 2 });
    expect(entry.unfilteredMessage).toEqual(message);
  });

  test("disabled keys are dropped and enabled keys come first", () => {
    const visibleKeys = ref(
      new Map<string[], boolean>([
        [["c"], true],
        [["a", "b"], false],
      ]),
    );
    const entry = new ComplexLogEntry(message, "c", 1, new Date(), "info", "stdout", "raw", visibleKeys);
    expect(entry.message).toEqual({ c: 2 });
  });

  test("enabled keys are ordered before remaining keys", () => {
    const visibleKeys = ref(new Map<string[], boolean>([[["a", "b"], true]]));
    const entry = new ComplexLogEntry(message, "c", 1, new Date(), "info", "stdout", "raw", visibleKeys);
    expect(Object.keys(entry.message)).toEqual(["a.b", "c"]);
  });
});

describe("SkippedLogsEntry", () => {
  function simple(id: number) {
    return new SimpleLogEntry(`m${id}`, "c", id, new Date(), "info", "stdout", `m${id}`);
  }

  test("renders the running skipped count and accumulates more", () => {
    const entry = new SkippedLogsEntry(new Date(), 3, simple(1), simple(2), async () => {});
    expect(entry.message).toBe("Skipped 3 entries");

    const newLast = simple(5);
    entry.addSkippedEntries(2, newLast);
    expect(entry.message).toBe("Skipped 5 entries");
    expect(entry.totalSkipped).toBe(5);
    expect(entry.lastSkippedLog).toBe(newLast);
  });
});
