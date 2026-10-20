import { describe, expect, test } from "vitest";
import { volumeColumns } from "./searchVolume";

const HOUR = 3_600_000;
const ns = (ms: number) => ms * 1e6;

describe("volumeColumns", () => {
  test("fills the empty buckets between the first match and now", () => {
    const columns = volumeColumns(
      [
        { ts: ns(0), level: "info", count: 4 },
        { ts: ns(3 * HOUR), level: "error", count: 2 },
      ],
      ns(HOUR),
      5 * HOUR + 10,
    );
    expect(columns.map((c) => c.total)).toEqual([4, 0, 0, 2, 0, 0]);
    expect(columns[0].start).toBe(0);
    expect(columns[5].start).toBe(5 * HOUR);
  });

  test("splits a bucket into error, warn and everything else", () => {
    const [column] = volumeColumns(
      [
        { ts: 0, level: "error", count: 1 },
        { ts: 0, level: "FATAL", count: 1 },
        { ts: 0, level: "warning", count: 3 },
        { ts: 0, level: "info", count: 5 },
        { ts: 0, level: "", count: 2 },
      ],
      ns(HOUR),
      0,
    );
    expect(column).toEqual({ start: 0, error: 2, warn: 3, other: 7, total: 12 });
  });

  test("draws nothing without buckets or a step", () => {
    expect(volumeColumns([], ns(HOUR), 0)).toEqual([]);
    expect(volumeColumns([{ ts: 0, level: "info", count: 1 }], 0, 0)).toEqual([]);
  });

  test("keeps the newest columns when the span is too wide to draw", () => {
    const columns = volumeColumns([{ ts: 0, level: "info", count: 1 }], ns(1000), 10_000 * 1000);
    expect(columns).toHaveLength(400);
    expect(columns.at(-1)!.start).toBe(10_000 * 1000);
  });
});
