import { describe, expect, test } from "vitest";
import { formatSpan, parseRange, parseSince, parseTime, rangeAround, timeRangeRoute } from "./timeRange";

const now = Date.parse("2026-10-02T15:42:10Z");

describe("parseSince", () => {
  test("a relative span resolves against now", () => {
    expect(parseSince("15m", now)).toEqual({
      kind: "since",
      since: new Date("2026-10-02T15:27:10Z"),
      relative: "15m",
    });
  });

  test("an absolute time", () => {
    expect(parseSince("2026-10-02T15:04:00Z", now)).toEqual({
      kind: "since",
      since: new Date("2026-10-02T15:04:00Z"),
    });
  });

  test("anything else is plain live", () => {
    expect(parseSince(undefined, now)).toEqual({ kind: "live" });
    expect(parseSince("soon", now)).toEqual({ kind: "live" });
    expect(parseSince(["15m"], now)).toEqual({ kind: "live" });
  });
});

describe("parseRange", () => {
  test("needs an end after the start", () => {
    expect(parseRange("2026-10-02T15:04:00Z", "2026-10-02T15:30:00Z")).toEqual({
      kind: "range",
      from: new Date("2026-10-02T15:04:00Z"),
      until: new Date("2026-10-02T15:30:00Z"),
    });
    expect(parseRange("2026-10-02T15:04:00Z", undefined)).toBeUndefined();
    expect(parseRange("2026-10-02T15:04:00Z", "2026-10-02T15:00:00Z")).toBeUndefined();
  });
});

describe("timeRangeRoute", () => {
  test("keeps a relative start relative in the URL", () => {
    expect(timeRangeRoute("abc", parseSince("1h", now))).toEqual({
      name: "/container/[id]",
      params: { id: "abc" },
      query: { since: "1h" },
    });
  });

  test("a range with an end is the historical route", () => {
    const range = parseRange("2026-10-02T15:04:00.000Z", "2026-10-02T15:30:00.000Z")!;
    expect(timeRangeRoute("abc", range)).toEqual({
      name: "/container/[id].time.[datetime]",
      params: { id: "abc", datetime: "2026-10-02T15:04:00.000Z" },
      query: { until: "2026-10-02T15:30:00.000Z" },
    });
  });
});

test("parseTime reads what people type", () => {
  expect(parseTime("9")).toEqual({ h: 9, m: 0, s: 0 });
  expect(parseTime("09:05")).toEqual({ h: 9, m: 5, s: 0 });
  expect(parseTime("21:05:30")).toEqual({ h: 21, m: 5, s: 30 });
  expect(parseTime("9:05 pm")).toEqual({ h: 21, m: 5, s: 0 });
  expect(parseTime("12:00 AM")).toEqual({ h: 0, m: 0, s: 0 });
  expect(parseTime("12:30pm")).toEqual({ h: 12, m: 30, s: 0 });
  expect(parseTime("24:00")).toBeUndefined();
  expect(parseTime("13 pm")).toBeUndefined();
  expect(parseTime("soon")).toBeUndefined();
});

test("rangeAround centers on the moment but never ends in the future", () => {
  const at = new Date("2026-10-02T03:17:59Z");
  expect(rangeAround(at, 60 * 60_000, now)).toEqual({
    kind: "range",
    from: new Date("2026-10-02T02:47:59Z"),
    until: new Date("2026-10-02T03:47:59Z"),
  });
  const recent = new Date(now - 60_000);
  const r = rangeAround(recent, 60 * 60_000, now);
  expect(r.kind === "range" && r.until.getTime()).toBe(now);
});

test("formatSpan", () => {
  expect(formatSpan(45_000)).toBe("45s");
  expect(formatSpan(26 * 60_000)).toBe("26m");
  expect(formatSpan(90 * 60_000)).toBe("1h 30m");
  expect(formatSpan(3 * 60 * 60_000)).toBe("3h");
  expect(formatSpan(52 * 60 * 60_000)).toBe("2d 4h");
});
