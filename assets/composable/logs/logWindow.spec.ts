/** @vitest-environment jsdom */
import { describe, expect, test, vi } from "vitest";
import { SimpleLogEntry, SkippedLogsEntry } from "@/models/LogEntry";
import { appendBatch } from "./logWindow";

function lines(from: number, count: number) {
  return Array.from(
    { length: count },
    (_, i) => new SimpleLogEntry(`line ${from + i}`, "c", from + i, new Date(from + i), "info", "stdout", ""),
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
