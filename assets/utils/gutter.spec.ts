import { describe, expect, test } from "vitest";
import { buildTicks, type AlertMeasurement } from "./gutter";

const at = (top: number, overrides: Partial<AlertMeasurement> = {}): AlertMeasurement => ({
  alertId: top,
  top,
  level: "error",
  headline: `alert at ${top}`,
  ...overrides,
});

describe("buildTicks", () => {
  test("maps pixel offsets to fractions of the content", () => {
    const ticks = buildTicks([at(0), at(500), at(1000)], 1000, 500);
    expect(ticks.map((t) => t.offset)).toEqual([0, 0.5, 1]);
  });

  test("orders ticks top to bottom regardless of input order", () => {
    const ticks = buildTicks([at(800), at(100), at(400)], 1000, 500);
    expect(ticks.map((t) => t.alertId)).toEqual([100, 400, 800]);
  });

  // A storm must not become a solid bar. The gutter's job is to say *where* to
  // scroll, and a solid bar says nothing.
  describe("merging", () => {
    test("collapses ticks closer than the minimum gap", () => {
      // 1000px of content in a 100px gutter: 10px of content = 1px of gutter,
      // so these three land within 3px of each other.
      const ticks = buildTicks([at(0), at(10), at(20)], 1000, 100);
      expect(ticks).toHaveLength(1);
      expect(ticks[0].count).toBe(3);
    });

    test("keeps ticks that are far enough apart", () => {
      const ticks = buildTicks([at(0), at(500)], 1000, 100);
      expect(ticks).toHaveLength(2);
      expect(ticks.every((t) => t.count === 1)).toBe(true);
    });

    test("a merged tick scrolls to the topmost alert of its cluster", () => {
      const ticks = buildTicks([at(0), at(10), at(20)], 1000, 100);
      expect(ticks[0].alertId).toBe(0);
    });

    test("a merged tick takes the worst level in the cluster", () => {
      const ticks = buildTicks([at(0, { level: "warn" }), at(10, { level: "error" })], 1000, 100);
      expect(ticks[0].level).toBe("error");
    });

    test("an unknown level never outranks a known one", () => {
      const ticks = buildTicks([at(0, { level: "error" }), at(10, { level: "mystery" })], 1000, 100);
      expect(ticks[0].level).toBe("error");
    });

    // The same alerts in a taller gutter have room to render separately.
    test("merging depends on gutter height, not content height", () => {
      const measurements = [at(0), at(10), at(20)];
      expect(buildTicks(measurements, 1000, 100)).toHaveLength(1);
      expect(buildTicks(measurements, 1000, 2000)).toHaveLength(3);
    });
  });

  describe("degenerate input", () => {
    // Dividing by an unmeasured height would stack every tick at the top,
    // which reads as real data rather than as "not laid out yet".
    test("returns nothing before the content has been laid out", () => {
      expect(buildTicks([at(100)], 0, 500)).toEqual([]);
      expect(buildTicks([at(100)], 1000, 0)).toEqual([]);
    });

    test("returns nothing when there are no alerts", () => {
      expect(buildTicks([], 1000, 500)).toEqual([]);
    });

    test("clamps an offset past the measured content", () => {
      const ticks = buildTicks([at(5000)], 1000, 500);
      expect(ticks[0].offset).toBe(1);
    });

    test("does not mutate the input", () => {
      const measurements = [at(500), at(100)];
      buildTicks(measurements, 1000, 500);
      expect(measurements.map((m) => m.top)).toEqual([500, 100]);
    });
  });
});
