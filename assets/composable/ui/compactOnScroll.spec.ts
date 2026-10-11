import { describe, expect, it } from "vitest";
import { nextCompact } from "./compactOnScroll";

describe("nextCompact", () => {
  const maxY = 2000;

  it("compacts when scrolling down past the offset", () => {
    expect(nextCompact({ y: 200, lastY: 150, maxY })).toBe(true);
  });

  it("stays full size when scrolling down near the top", () => {
    expect(nextCompact({ y: 60, lastY: 20, maxY })).toBe(false);
  });

  it("expands when scrolling up", () => {
    expect(nextCompact({ y: 500, lastY: 600, maxY })).toBe(false);
  });

  it("ignores movements under the jitter threshold", () => {
    expect(nextCompact({ y: 504, lastY: 500, maxY })).toBeNull();
  });

  it("ignores rubber-banding past either end", () => {
    expect(nextCompact({ y: -30, lastY: 0, maxY })).toBeNull();
    expect(nextCompact({ y: 2040, lastY: 1990, maxY })).toBeNull();
  });
});
