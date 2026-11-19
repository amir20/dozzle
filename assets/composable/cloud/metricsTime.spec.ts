/**
 * @vitest-environment jsdom
 */
import { describe, expect, test } from "vitest";
import { hoverSpan, spanMiddle } from "./metricsTime";

describe("hoverSpan", () => {
  test("one point per bar when the chart draws them all", () => {
    expect(hoverSpan(110, { index: 7, bars: 110 })).toEqual([7, 8]);
  });

  // BarChart folds 300 samples into runs of ceil(300 / 110) = 3.
  test("a folded bar covers the same run BarChart averaged", () => {
    expect(hoverSpan(300, { index: 0, bars: 100 })).toEqual([0, 3]);
    expect(hoverSpan(300, { index: 99, bars: 100 })).toEqual([297, 300]);
  });

  test("the last run stops at the end of the series", () => {
    expect(hoverSpan(10, { index: 3, bars: 4 })).toEqual([9, 10]);
  });
});

describe("spanMiddle", () => {
  test("names the middle point of a run", () => {
    expect(spanMiddle([0, 3])).toBe(1);
    expect(spanMiddle([4, 5])).toBe(4);
  });
});
