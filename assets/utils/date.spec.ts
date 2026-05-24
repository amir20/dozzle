/**
 * @vitest-environment jsdom
 */
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { formatDuration, toRelativeTime } from "./date";

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
