/**
 * @vitest-environment jsdom
 */
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { dayBucket, formatDuration, groupByDay, toElapsed, toRelativeTime } from "./date";

describe("formatDuration", () => {
  // Assertions stay tolerant so they pass whether Intl.DurationFormat is present
  // (narrow style emits seconds) or the manual fallback is used (drops seconds).
  test("hours and minutes", () => {
    const result = formatDuration(3661, "en");
    expect(result).toContain("1h");
    expect(result).toContain("1m");
  });

  test("minutes and seconds", () => {
    const result = formatDuration(90, "en");
    expect(result).toContain("1m");
    expect(result).toContain("30s");
  });

  test("seconds only", () => {
    expect(formatDuration(45, "en")).toContain("45s");
  });
});

describe("toRelativeTime", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-15T00:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test("past days", () => {
    expect(toRelativeTime(new Date("2026-01-13T00:00:00Z"), "en")).toBe("2 days ago");
  });

  test("past hour", () => {
    expect(toRelativeTime(new Date("2026-01-14T23:00:00Z"), "en")).toBe("1 hour ago");
  });

  test("future", () => {
    expect(toRelativeTime(new Date("2026-01-17T00:00:00Z"), "en")).toBe("in 2 days");
  });
});

describe("toElapsed", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-15T00:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test("whole largest unit, no direction", () => {
    expect(toElapsed(new Date("2026-01-14T23:51:30Z"), "en")).toBe("8 minutes");
    expect(toElapsed(new Date("2026-01-13T00:00:00Z"), "en")).toBe("2 days");
  });

  test("just started", () => {
    expect(toElapsed(new Date("2026-01-15T00:00:00Z"), "en")).toBe("0 seconds");
  });
});

describe("dayBucket", () => {
  // Local time on purpose: days are the reader's days. Tests run with TZ=UTC.
  const now = new Date(2026, 10, 2, 9, 0);
  const label = (d: Date, locale = "en") => dayBucket(d, locale, now)?.label;

  test("today and yesterday", () => {
    expect(label(new Date(2026, 10, 2, 0, 5))).toBe("Today");
    expect(label(new Date(2026, 10, 1, 23, 59))).toBe("Yesterday");
  });

  test("weekday within the past week", () => {
    expect(label(new Date(2026, 9, 29, 12))).toBe("Thursday");
  });

  test("one last-week bucket, then nothing", () => {
    expect(dayBucket(new Date(2026, 9, 26, 12), "en", now)).toEqual({ key: -7, label: "Last week" });
    expect(dayBucket(new Date(2026, 9, 20, 12), "en", now)).toEqual({ key: -7, label: "Last week" });
    expect(dayBucket(new Date(2026, 9, 19, 12), "en", now)).toBeUndefined();
  });

  test("a future timestamp from clock skew files under today", () => {
    expect(label(new Date(2026, 10, 3, 0, 1))).toBe("Today");
  });

  test("translated by Intl", () => {
    expect(label(new Date(2026, 10, 1, 12), "de")).toBe("Gestern");
  });
});

describe("groupByDay", () => {
  test("keeps order within each day", () => {
    const now = new Date(2026, 10, 2, 9, 0);
    const items = [new Date(2026, 10, 2, 8), new Date(2026, 10, 2, 1), new Date(2026, 10, 1, 22)];
    const groups = groupByDay(items, (d) => d, "en", now);
    expect(groups.map((g) => g.label)).toEqual(["Today", "Yesterday"]);
    expect(groups[0].items).toEqual([items[0], items[1]]);
  });

  test("merges last week into one group and drops older", () => {
    const now = new Date(2026, 10, 2, 9, 0);
    const items = [new Date(2026, 9, 25), new Date(2026, 9, 21), new Date(2026, 9, 1)];
    const groups = groupByDay(items, (d) => d, "en", now);
    expect(groups.map((g) => g.label)).toEqual(["Last week"]);
    expect(groups[0].items).toEqual([items[0], items[1]]);
  });
});
