/** @vitest-environment jsdom */
import { describe, expect, test } from "vitest";
import { narrowedLevels, routeKind, toViewLogLines } from "./viewContext";
import { SimpleLogEntry, SkippedLogsEntry, type Level } from "@/models/LogEntry";

describe("routeKind", () => {
  test("names the shape of the page, not the route", () => {
    expect(routeKind("/container/[id]")).toBe("container");
    expect(routeKind("/host/[id]")).toBe("host");
    expect(routeKind("/merged/[ids]")).toBe("merged");
    expect(routeKind("/stack/[name]")).toBe("stack");
    expect(routeKind("/notifications")).toBe("notifications");
  });

  test("collapses the time-travel route onto the container it shows", () => {
    expect(routeKind("/container/[id].time.[datetime]")).toBe("container");
  });

  test("treats home and unnamed routes as the index", () => {
    expect(routeKind("/")).toBe("index");
    expect(routeKind(undefined)).toBe("index");
    expect(routeKind(null)).toBe("index");
    expect(routeKind(Symbol("anon"))).toBe("index");
  });
});

describe("narrowedLevels", () => {
  const all: Level[] = ["info", "debug", "warn", "error", "fatal", "trace", "unknown"];

  test("reports nothing when every level is shown, because that is not a filter", () => {
    expect(narrowedLevels(new Set(all))).toBeUndefined();
  });

  test("reports the narrowed set in a stable order", () => {
    expect(narrowedLevels(new Set(["error", "warn"] as Level[]))).toEqual(["warn", "error"]);
  });

  test("treats an empty set as no filter rather than as hiding everything", () => {
    expect(narrowedLevels(new Set())).toBeUndefined();
  });
});

describe("toViewLogLines", () => {
  const at = (i: number) => new Date(Date.UTC(2026, 0, 1, 0, 0, i));
  const simple = (i: number, message = `line ${i}`) =>
    new SimpleLogEntry(message, "abc", i, at(i), "info", "stdout", message);

  test("keeps the window in reading order", () => {
    const lines = toViewLogLines([simple(1), simple(2), simple(3)], false);
    expect(lines.map((l) => l.message)).toEqual(["line 1", "line 2", "line 3"]);
    expect(lines[0].timestamp).toBe(at(1).toISOString());
  });

  test("drops the oldest lines rather than the ones being read", () => {
    const entries = Array.from({ length: 400 }, (_, i) => simple(i));
    const lines = toViewLogLines(entries, false);
    expect(lines).toHaveLength(150);
    expect(lines.at(-1)?.message).toBe("line 399");
  });

  test("truncates a line nobody would read in full", () => {
    const [line] = toViewLogLines([simple(1, "x".repeat(5000))], false);
    expect(line.message).toHaveLength(2001);
    expect(line.message.endsWith("\u2026")).toBe(true);
  });

  test("leaves out the placeholders that stand for logs rather than being logs", () => {
    const skipped = new SkippedLogsEntry(at(0), 10, simple(0), simple(1), async () => {});
    const entries = [skipped, simple(2)];
    expect(toViewLogLines(entries, false).map((l) => l.message)).toEqual(["line 2"]);
  });

  test("names the container only where a line would be ambiguous without it", () => {
    expect(toViewLogLines([simple(1)], false)[0].containerId).toBeUndefined();
    expect(toViewLogLines([simple(1)], true)[0].containerId).toBe("abc");
  });

  test("strips the colour the terminal put there", () => {
    const [line] = toViewLogLines([simple(1, "\u001b[31mred\u001b[0m")], false);
    expect(line.message).toBe("red");
  });
});
