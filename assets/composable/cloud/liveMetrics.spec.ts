import { describe, expect, test } from "vitest";
import { containerLiveSeries, hostLiveSeries, liveMinutes } from "./liveMetrics";
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

  test("a host sums its running containers per sample, aligned at the newest", () => {
    const host = { id: "h1", name: "box", nCPU: 2, memTotal: 4096 };
    const series = hostLiveSeries(host, [
      container(),
      container({ id: "b2", statsHistory: [stat(20, 50)], sampledStats: 1 }),
      container({ id: "c3", state: "exited", statsHistory: [stat(99, 999)], sampledStats: 1 }),
      container({ id: "d4", host: "h2" }),
    ]);
    expect(series.points).toHaveLength(300);
    expect(series.points.at(-1)).toEqual({ cpu: 50, memoryUsage: 250 });
    expect(series.points.at(-2)).toEqual({ cpu: 20, memoryUsage: 100 });
    expect(series.sampledFrom).toBe(298);
    expect(series.memoryLimit).toBe(4096);
  });

  test("the note's minutes round up and never say zero", () => {
    const one = containerLiveSeries(container(), 1);
    expect(liveMinutes([one])).toBe(1);
    const many = containerLiveSeries(container({ sampledStats: 3, statsHistory: new Array(150).fill(stat(1, 1)) }), 1);
    expect(liveMinutes([{ ...many, sampledFrom: 0 }])).toBe(3);
  });
});
