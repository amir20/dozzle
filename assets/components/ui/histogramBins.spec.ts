import { describe, expect, test } from "vitest";
import { distribution, quantile } from "./histogramBins";

const of = (...values: number[]) => new Float64Array(values);

describe("quantile", () => {
  test("interpolates between ranks like quantile_cont", () => {
    expect(quantile(of(1, 2, 3, 4), 0.5)).toBe(2.5);
    expect(quantile(of(10, 20, 30), 0.5)).toBe(20);
    expect(quantile(of(0, 100), 0.95)).toBe(95);
  });
});

describe("distribution", () => {
  test("returns null for no values", () => {
    expect(distribution(of())).toBeNull();
  });

  test("sorts numerically, not as strings", () => {
    const d = distribution(of(100, 9, 20))!;
    expect(d.min).toBe(9);
    expect(d.max).toBe(100);
    expect(d.p50).toBe(20);
  });

  test("counts every value exactly once, max included", () => {
    const values = of(...Array.from({ length: 500 }, (_, i) => 200 + ((i * 37) % 100)), 787);
    const d = distribution(values)!;
    expect(d.bins.reduce((sum, b) => sum + b.count, 0)).toBe(values.length);
    expect(d.bins[0].start).toBe(d.min);
    expect(d.bins.at(-1)!.end).toBeCloseTo(d.max);
    expect(d.bins.at(-1)!.count).toBe(1);
  });

  test("keeps the bulk spread over several bins when there is a long tail", () => {
    const values = of(...Array.from({ length: 80 }, (_, i) => 225 + i), 787);
    const d = distribution(values)!;
    expect(d.bins.filter((b) => b.count > 0 && b.end <= 305).length).toBeGreaterThan(3);
  });

  test("a constant column is one bin", () => {
    const d = distribution(of(5, 5, 5))!;
    expect(d.bins).toEqual([{ start: 5, end: 5, count: 3 }]);
  });

  test("integer bins are whole-number and inclusive", () => {
    const d = distribution(of(0, 1, 1, 2, 3, 3, 3, 4), true)!;
    expect(d.bins.map((b) => [b.start, b.end, b.count])).toEqual([
      [0, 0, 1],
      [1, 1, 2],
      [2, 2, 1],
      [3, 3, 3],
      [4, 4, 1],
    ]);
  });
});
