import { describe, expect, test, vi } from "vitest";
import { Container, emptyStat, type Stat } from "./Container";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [{ name: "localhost", id: "localhost" }] },
  withBase: (path: string) => path,
}));

function makeContainer(
  overrides: { labels?: Record<string, string>; image?: string; command?: string; stats?: Stat[] } = {},
) {
  return new Container(
    "id-1",
    new Date(),
    new Date(),
    new Date(),
    overrides.image ?? "image",
    "display-name",
    overrides.command ?? "command",
    "localhost",
    overrides.labels ?? {},
    "running",
    0,
    0,
    overrides.stats ?? [],
  );
}

function makeStat(partial: Partial<Stat> = {}): Stat {
  return { ...emptyStat(), ...partial };
}

describe("Container.namespace", () => {
  test("prefers dev.dozzle.group over all others", () => {
    const c = makeContainer({
      labels: {
        "dev.dozzle.group": "g",
        "coolify.projectName": "c",
        "com.docker.stack.namespace": "s",
        "com.docker.compose.project": "p",
      },
    });
    expect(c.namespace).toBe("g");
  });

  test("falls back through coolify, stack, then compose", () => {
    expect(makeContainer({ labels: { "coolify.projectName": "c" } }).namespace).toBe("c");
    expect(makeContainer({ labels: { "com.docker.stack.namespace": "s" } }).namespace).toBe("s");
    expect(makeContainer({ labels: { "com.docker.compose.project": "p" } }).namespace).toBe("p");
  });

  test("undefined when no grouping label is present", () => {
    expect(makeContainer().namespace).toBeUndefined();
  });
});

describe("Container.name", () => {
  test("non-swarm returns the constructor name and respects the setter", () => {
    const c = makeContainer();
    expect(c.name).toBe("display-name");
    c.name = "renamed";
    expect(c.name).toBe("renamed");
  });

  test("swarm strips task id and node id from the task name", () => {
    const c = makeContainer({
      labels: {
        "com.docker.swarm.service.id": "svc",
        "com.docker.swarm.task.name": "api.1.t1n0d3",
        "com.docker.swarm.task.id": "t1n0d3",
        "com.docker.swarm.node.id": "node99",
      },
    });
    expect(c.isSwarm).toBe(true);
    expect(c.name).toBe("api.1");
    expect(c.swarmId).toBe("t1n0d3");
  });
});

describe("Container.storageKey", () => {
  test("combines stripped image with command", () => {
    expect(makeContainer({ image: "nginx:1.25", command: "run" }).storageKey).toBe("nginx:run");
  });
});

describe("Container.hostLabel", () => {
  test("resolves the host name from config", () => {
    expect(makeContainer().hostLabel).toBe("localhost");
  });
});

describe("Container stats history", () => {
  test("pads history to 300 and seeds latest stat", () => {
    const a = makeStat({ cpu: 1 });
    const b = makeStat({ cpu: 2 });
    const c = makeContainer({ stats: [a, b] });
    expect(c.statsHistory).toHaveLength(300);
    expect(c.statsHistory.at(-1)).toEqual(b);
    expect(c.statsHistory.at(-2)).toEqual(a);
    expect(c.stat).toEqual(b);
  });

  test("empty stats seed an empty stat", () => {
    const c = makeContainer();
    expect(c.statsHistory).toHaveLength(300);
    expect(c.stat).toEqual(emptyStat());
  });
});

describe("Container.updateStat", () => {
  test("applies EMA (alpha 0.2) to cpu/memory and passes totals through", () => {
    const c = makeContainer();
    c.updateStat(makeStat({ cpu: 10, memory: 50, memoryUsage: 100, networkRxTotal: 5, diskWriteTotal: 8 }));

    expect(c.stat.cpu).toBe(10);
    expect(c.movingAverage.cpu).toBeCloseTo(2, 10);
    expect(c.movingAverage.memory).toBeCloseTo(10, 10);
    expect(c.movingAverage.memoryUsage).toBeCloseTo(20, 10);
    // totals are not averaged
    expect(c.movingAverage.networkRxTotal).toBe(5);
    expect(c.movingAverage.diskWriteTotal).toBe(8);
  });

  test("EMA folds in the previous moving average on each update", () => {
    const c = makeContainer();
    c.updateStat(makeStat({ cpu: 10, memory: 50, memoryUsage: 100 }));
    c.updateStat(makeStat({ cpu: 10, memory: 50, memoryUsage: 100 }));

    expect(c.movingAverage.cpu).toBeCloseTo(3.6, 10); // 0.2*10 + 0.8*2
    expect(c.movingAverage.memory).toBeCloseTo(18, 10); // 0.2*50 + 0.8*10
    expect(c.movingAverage.memoryUsage).toBeCloseTo(36, 10); // 0.2*100 + 0.8*20
  });

  test("history stays capped at 300 with the newest stat last", () => {
    const c = makeContainer();
    const latest = makeStat({ cpu: 42 });
    c.updateStat(latest);
    expect(c.statsHistory).toHaveLength(300);
    expect(c.statsHistory.at(-1)).toEqual(latest);
  });
});
