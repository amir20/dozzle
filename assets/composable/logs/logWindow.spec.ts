/** @vitest-environment jsdom */
import { describe, expect, test, vi } from "vitest";
import { SimpleLogEntry, SkippedLogsEntry } from "@/models/LogEntry";
import { appendBatch, newerThanOnScreen, newestOnScreen, notOnScreen } from "./logWindow";

function lines(from: number, count: number, container = "c") {
  return Array.from(
    { length: count },
    (_, i) => new SimpleLogEntry(`line ${from + i}`, container, from + i, new Date(from + i), "info", "stdout", ""),
  );
}

const loadSkipped = vi.fn(async () => {});

describe("appendBatch", () => {
  test("appends when the batch fits", () => {
    const result = appendBatch(lines(0, 2), lines(2, 3), { maxLogs: 10, paused: false, loadSkipped });
    expect(result.map((m) => m.id)).toEqual([0, 1, 2, 3, 4]);
  });

  test("drops the oldest lines on overflow", () => {
    const result = appendBatch(lines(0, 8), lines(8, 4), { maxLogs: 10, paused: false, loadSkipped });
    expect(result.map((m) => m.id)).toEqual([2, 3, 4, 5, 6, 7, 8, 9, 10, 11]);
  });

  test("keeps only the newest half when the batch alone is over half the window", () => {
    const result = appendBatch(lines(0, 8), lines(8, 6), { maxLogs: 10, paused: false, loadSkipped });
    expect(result.map((m) => m.id)).toEqual([9, 10, 11, 12, 13]);
  });

  test("collapses an overflowing batch into a skipped entry while paused", () => {
    const messages = lines(0, 8);
    const result = appendBatch(messages, lines(8, 4), { maxLogs: 10, paused: true, loadSkipped });
    expect(result).toHaveLength(9);
    const skipped = result.at(-1) as SkippedLogsEntry;
    expect(skipped).toBeInstanceOf(SkippedLogsEntry);
    expect(skipped.message).toBe("Skipped 4 entries");
    expect(skipped.firstSkipped.id).toBe(8);
  });

  test("grows the existing skipped entry in place", () => {
    const first = appendBatch(lines(0, 8), lines(8, 4), { maxLogs: 10, paused: true, loadSkipped });
    const second = appendBatch(first, lines(12, 3), { maxLogs: 10, paused: true, loadSkipped });
    expect(second).toBe(first);
    expect((second.at(-1) as SkippedLogsEntry).message).toBe("Skipped 7 entries");
  });
});

describe("newerThanOnScreen", () => {
  test("drops the replayed tail and keeps what follows it", () => {
    const keep = newerThanOnScreen(lines(0, 10));
    expect(
      lines(5, 10)
        .filter(keep)
        .map((m) => m.id),
    ).toEqual([10, 11, 12, 13, 14]);
  });

  test("keeps a different line stamped the same millisecond as the last one seen", () => {
    const keep = newerThanOnScreen(lines(0, 3));
    const twin = new SimpleLogEntry("other", "c", 99, new Date(2), "info", "stdout", "");
    expect(keep(lines(2, 1)[0])).toBe(false);
    expect(keep(twin)).toBe(true);
  });

  test("cuts off each container at its own last line", () => {
    const keep = newerThanOnScreen([...lines(0, 5, "a"), ...lines(0, 10, "b")]);
    expect(
      lines(3, 4, "a")
        .filter(keep)
        .map((m) => m.id),
    ).toEqual([5, 6]);
    expect(lines(3, 4, "b").filter(keep)).toEqual([]);
    expect(lines(0, 2, "new").filter(keep)).toHaveLength(2);
  });

  test("counts lines collapsed into a skipped entry as seen", () => {
    const skipped = new SkippedLogsEntry(new Date(), 5, lines(5, 1)[0], lines(9, 1)[0], loadSkipped);
    const keep = newerThanOnScreen([...lines(0, 5), skipped]);
    expect(
      lines(7, 5)
        .filter(keep)
        .map((m) => m.id),
    ).toEqual([10, 11]);
  });
});

describe("notOnScreen", () => {
  test("drops backfilled lines already in the view", () => {
    const keep = notOnScreen(lines(5, 5));
    expect(
      lines(0, 8)
        .filter(keep)
        .map((m) => m.id),
    ).toEqual([0, 1, 2, 3, 4]);
  });
});

describe("newestOnScreen", () => {
  test("reads through a skipped entry and ignores rows that are not lines", () => {
    const skipped = new SkippedLogsEntry(new Date(10_000), 5, lines(5, 1)[0], lines(9, 1)[0], loadSkipped);
    expect(newestOnScreen([...lines(0, 5), skipped])).toBe(9);
  });
});
