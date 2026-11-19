import { describe, expect, test } from "vitest";
import { aggregateHosts, hostStats, mergeTop } from "./hostMetrics";

describe("hostMetrics", () => {
  test("across hosts CPU is the average host and the rest add up, bucket by bucket", () => {
    const merged = aggregateHosts([
      [{ ts: 1, cpu: 10, memoryUsage: 100, cpuMax: 20, networkRx: 5 }],
      [
        { ts: 1, cpu: 30, memoryUsage: 50, memoryUsageMax: 80 },
        { ts: 2, cpu: 40, memoryUsage: 60 },
      ],
    ]);
    expect(merged).toEqual([
      // A host without a max counts with its average.
      {
        ts: 1,
        cpu: 20,
        cpuMax: 25,
        memoryUsage: 150,
        memoryUsageMax: 180,
        networkRx: 5,
        networkTx: undefined,
        diskRead: undefined,
        diskWrite: undefined,
      },
      {
        ts: 2,
        cpu: 40,
        cpuMax: undefined,
        memoryUsage: 60,
        memoryUsageMax: undefined,
        networkRx: undefined,
        networkTx: undefined,
        diskRead: undefined,
        diskWrite: undefined,
      },
    ]);
  });

  test("a host's stats take the peak from the max and add I/O directions", () => {
    const stats = hostStats([
      { ts: 1, cpu: 10, memoryUsage: 1, cpuMax: 90, networkRx: 1, networkTx: 2 },
      { ts: 2, cpu: 30, memoryUsage: 3 },
    ]);
    expect(stats.cpu).toEqual({ avg: 20, peak: 90 });
    expect(stats.memory).toEqual({ avg: 2, peak: 3 });
    expect(stats.network).toEqual({ avg: 1.5, peak: 3 });
  });

  test("top lists merge heaviest first and keep the host", () => {
    const merged = mergeTop(
      [
        { hostId: "a", list: [{ name: "x", avg: 1, peak: 2 }] },
        {
          hostId: "b",
          list: [
            { name: "y", avg: 5, peak: 6 },
            { name: "z", avg: 3, peak: 4 },
          ],
        },
      ],
      2,
    );
    expect(merged.map((m) => `${m.hostId}/${m.name}`)).toEqual(["b/y", "b/z"]);
  });
});
