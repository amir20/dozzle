import { describe, expect, test } from "vitest";
import { containerLiveSeries, liveMinutes } from "./liveMetrics";
import type { Container } from "@/models/Container";

const stat = (cpu: number, memoryUsage: number) => ({ cpu, memory: 0, memoryUsage });

function container(over: Partial<Record<string, unknown>> = {}) {
  return {
    id: "a1",
    name: "api",
    host: "h1",
    state: "running",
    cpuLimit: 0,
    memoryLimit: 0,
    statsHistory: [stat(0, 0), stat(40, 100), stat(80, 200)],
    sampledStats: 2,
    ...over,
  } as unknown as Container;
}

describe("liveMetrics", () => {
  test("a container's CPU reads against its limit, or the host's cores without one", () => {
    expect(containerLiveSeries(container(), 4).points.map((p) => p.cpu)).toEqual([0, 10, 20]);
    expect(containerLiveSeries(container({ cpuLimit: 2 }), 4).points.map((p) => p.cpu)).toEqual([0, 20, 40]);
  });

  test("the padding in front of the samples is marked, and memory reads against a limit when set", () => {
    const series = containerLiveSeries(container({ memoryLimit: 1000 }), 1);
    expect(series.sampledFrom).toBe(1);
    expect(series.memoryLimit).toBe(1000);
    expect(containerLiveSeries(container(), 1).memoryLimit).toBe(0);
  });

  test("the note's minutes round up and never say zero", () => {
    const one = containerLiveSeries(container(), 1);
    expect(liveMinutes([one])).toBe(1);
    const many = containerLiveSeries(container({ sampledStats: 3, statsHistory: new Array(150).fill(stat(1, 1)) }), 1);
    expect(liveMinutes([{ ...many, sampledFrom: 0 }])).toBe(3);
  });
});
