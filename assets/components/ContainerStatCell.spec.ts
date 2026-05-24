/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import ContainerStatCell from "./ContainerStatCell.vue";
import { Container, emptyStat, type Stat } from "@/models/Container";
import type { Host } from "@/stores/hosts";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [{ name: "localhost", id: "localhost" }] },
  withBase: (path: string) => path,
}));

function makeStat(partial: Partial<Stat> = {}): Stat {
  return { ...emptyStat(), ...partial };
}

function makeContainer(lastStat: Partial<Stat>, cpuLimit = 0): Container {
  return new Container(
    "id-1",
    new Date(),
    new Date(),
    new Date(),
    "image",
    "name",
    "command",
    "localhost",
    {},
    "running",
    cpuLimit,
    0,
    [makeStat(lastStat)],
  );
}

function host(nCPU?: number): Host {
  return { id: "localhost", name: "localhost", nCPU } as unknown as Host;
}

function mountCell(props: { container: Container; type: "cpu" | "mem"; host: Host; mode?: "chart" | "progress" }) {
  return mount(ContainerStatCell, {
    props: { mode: "progress", ...props },
    global: { stubs: { BarChart: true } },
  });
}

describe("<ContainerStatCell /> cpu", () => {
  test("normalizes by cpuLimit when set", () => {
    const wrapper = mountCell({ container: makeContainer({ cpu: 100 }, 2), type: "cpu", host: host(8) });
    expect(wrapper.find(".tabular-nums").text()).toBe("50%");
    expect(wrapper.find("progress").attributes("value")).toBe("50");
    expect(wrapper.find("progress").classes()).toContain("progress-success");
  });

  test("falls back to host nCPU when no cpuLimit", () => {
    const wrapper = mountCell({ container: makeContainer({ cpu: 360 }, 0), type: "cpu", host: host(4) });
    expect(wrapper.find(".tabular-nums").text()).toBe("90%");
    expect(wrapper.find("progress").classes()).toContain("progress-warning");
  });

  test("falls back to a single core and clamps at 100%", () => {
    const wrapper = mountCell({ container: makeContainer({ cpu: 200 }, 0), type: "cpu", host: host(undefined) });
    expect(wrapper.find(".tabular-nums").text()).toBe("100%");
    expect(wrapper.find("progress").classes()).toContain("progress-error");
  });
});

describe("<ContainerStatCell /> memory", () => {
  test("shows absolute usage and percentage-based color", () => {
    const wrapper = mountCell({
      container: makeContainer({ memory: 40, memoryUsage: 1500 }),
      type: "mem",
      host: host(4),
    });
    expect(wrapper.find(".tabular-nums").text()).toBe("1.46 KB");
    expect(wrapper.find("progress").attributes("value")).toBe("40");
    expect(wrapper.find("progress").classes()).toContain("progress-success");
  });
});

describe("<ContainerStatCell /> color thresholds", () => {
  test.each([
    [50, "bg-success"],
    [70, "bg-secondary"],
    [90, "bg-warning"],
    [95, "bg-error"],
  ])("memory %i%% -> %s", (memory, expected) => {
    const wrapper = mountCell({ container: makeContainer({ memory }), type: "mem", host: host(4) });
    expect((wrapper.vm as unknown as { barClass: string }).barClass).toBe(expected);
  });
});

describe("<ContainerStatCell /> chart data", () => {
  test("cpu series is normalized by cores", () => {
    const wrapper = mountCell({ container: makeContainer({ cpu: 100 }, 4), type: "cpu", host: host(8) });
    const chartData = (wrapper.vm as unknown as { chartData: { percent: number; value: number }[] }).chartData;
    expect(chartData).toHaveLength(300);
    expect(chartData.at(-1)).toEqual({ percent: 25, value: 100 });
  });

  test("memory series uses percent and absolute usage", () => {
    const wrapper = mountCell({
      container: makeContainer({ memory: 30, memoryUsage: 2048 }),
      type: "mem",
      host: host(4),
    });
    const chartData = (wrapper.vm as unknown as { chartData: { percent: number; value: number }[] }).chartData;
    expect(chartData.at(-1)).toEqual({ percent: 30, value: 2048 });
  });
});
