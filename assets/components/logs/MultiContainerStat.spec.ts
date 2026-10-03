/**
 * @vitest-environment jsdom
 */
import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { createI18n } from "vue-i18n";
import { defineComponent, h, nextTick, ref } from "vue";
import { Container, Stat } from "@/models/Container";
import MultiContainerStat from "./MultiContainerStat.vue";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { hosts: [{ id: "host1", name: "host1", nCPU: 4, memTotal: 1000 }], base: "" },
  withBase: (path: string) => path,
}));

// Capture recalculate() across both chart instances.
const recalculate = vi.fn();
const BarChartStub = defineComponent({
  name: "BarChart",
  props: ["chartData", "barClass"],
  setup(_, { expose }) {
    expose({ recalculate });
    return () => h("div", { class: "bar-chart-stub" });
  },
});

const i18n = createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false, messages: { en: {} } });

function stat(cpu: number): Stat {
  return {
    cpu,
    memory: cpu,
    memoryUsage: cpu,
    networkRxTotal: 0,
    networkTxTotal: 0,
    diskReadTotal: 0,
    diskWriteTotal: 0,
  };
}

function makeContainer(id: string, cpu: number, cpuLimit = 0): Container {
  const stats = Array.from({ length: 10 }, () => stat(cpu));
  const now = new Date();
  return new Container(id, now, now, now, "img", id, "cmd", "host1", {}, "running", cpuLimit, 0, stats);
}

function lastCpu(containers: Container[]) {
  const wrapper = mount(MultiContainerStat as any, {
    props: { containers },
    global: { plugins: [i18n], stubs: { BarChart: BarChartStub, IOCard: true } },
  });
  return () => wrapper.findAllComponents(BarChartStub)[0].props("chartData").at(-1).value;
}

describe("<MultiContainerStat />", () => {
  test("recalculates the charts when the container changes", async () => {
    // Mirror production: a parent holding a ref re-renders with a fresh
    // [container] array on switch. (VueTestUtils setProps cannot trigger this
    // because Container instances carry refs, which defeats prop change
    // detection on a direct prop assignment.)
    const current = ref<Container>(makeContainer("a", 10));
    const Parent = defineComponent({
      setup: () => () => h(MultiContainerStat as any, { containers: [current.value] }),
    });
    mount(Parent, {
      global: { plugins: [i18n], stubs: { BarChart: BarChartStub, IOCard: true } },
    });
    await flushPromises();
    recalculate.mockClear();

    // Switching containers replaces the whole stats series; the parent must
    // force the cached charts to fully recompute.
    current.value = makeContainer("b", 90);
    await nextTick();
    await flushPromises();

    expect(recalculate).toHaveBeenCalled();
  });

  test("leaves a stopped container's last sample out of the total", async () => {
    vi.useFakeTimers();
    const stopped = makeContainer("b", 400);
    stopped.state = "exited";
    // One core of four in use; the stopped one's four cores are history.
    const cpu = lastCpu([makeContainer("a", 100), stopped]);
    expect(cpu()).toBe(25);
    await vi.advanceTimersByTimeAsync(1100);
    expect(cpu()).toBe(25);
    vi.useRealTimers();
  });

  test("measures cpu against the cores the containers can use", async () => {
    vi.useFakeTimers();
    // Two half-core containers, both pegged, are using all of one core.
    const cpu = lastCpu([makeContainer("a", 50, 0.5), makeContainer("b", 50, 0.5)]);
    await vi.advanceTimersByTimeAsync(1100);
    expect(cpu()).toBe(100);
    vi.useRealTimers();
  });
});
