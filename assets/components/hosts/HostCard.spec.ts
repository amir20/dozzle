/**
 * @vitest-environment jsdom
 */
import { createTestingPinia } from "@pinia/testing";
import { mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { createI18n } from "vue-i18n";
import { defineComponent, h } from "vue";
import { Container, Stat } from "@/models/Container";
import type { Host } from "@/stores/hosts";
import HostCard from "./HostCard.vue";

// @ts-ignore
import EventSource from "eventsourcemock";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { hosts: [], base: "", version: "test" },
  withBase: (path: string) => path,
}));

const MetricCardStub = defineComponent({
  name: "MetricCard",
  props: ["label", "value"],
  setup(_, { expose }) {
    expose({ recalculate: () => {} });
    return () => h("div");
  },
});

const i18n = createI18n({
  legacy: false,
  locale: "en",
  missingWarn: false,
  fallbackWarn: false,
  messages: {
    en: {
      label: {
        disk: "Disk",
        load: "Load",
        uptime: "Uptime",
        network: "Network",
        "since-boot": "since boot",
        "load-avg": "1m · 5m · 15m",
        "used-percent": "{percent}% used",
      },
    },
  },
});

const host: Host = {
  id: "host1",
  name: "host1",
  nCPU: 4,
  memTotal: 0,
  type: "local",
  endpoint: "local",
  available: true,
  dockerVersion: "",
  agentVersion: "",
};

function stat(cpu: number): Stat {
  return {
    cpu,
    memory: 0,
    memoryUsage: 0,
    networkRxTotal: 0,
    networkTxTotal: 0,
    diskReadTotal: 0,
    diskWriteTotal: 0,
  };
}

function makeContainer(id: string, cpu: number, cpuLimit: number): Container {
  const now = new Date();
  return new Container(id, now, now, now, "img", id, "cmd", "host1", {}, "running", cpuLimit, 0, [stat(cpu)]);
}

describe("<HostCard />", () => {
  test("totals CPU against the host's cores, even for containers with a limit", () => {
    // Raw stats are percent of one core: 50 is half a core. On a 4-core host, an
    // unlimited container at 1 core and a `cpus: 0.5` one at 0.5 core use 1.5 of
    // 4 cores, 37.5%. Dividing the limited one by its own limit reported 125%.
    global.EventSource = EventSource;
    const wrapper = mount(HostCard, {
      props: { host },
      global: {
        plugins: [
          i18n,
          createTestingPinia({
            createSpy: vi.fn,
            initialState: {
              container: { containers: [makeContainer("a", 100, 0), makeContainer("b", 50, 0.5)] },
            },
          }),
        ],
        stubs: { MetricCard: MetricCardStub, HostIcon: true },
      },
    });

    const cpu = wrapper.findAllComponents(MetricCardStub).find((card) => card.props("label") === "CPU")!;
    expect(cpu.props("value")).toBeCloseTo(37.5);
  });

  test("renders host metric tiles when the host reports them", () => {
    global.EventSource = EventSource;
    const hostWithMetrics: Host = {
      ...host,
      metricsAvailable: true,
      memUsed: 1,
      load1: 0.5,
      load5: 0.4,
      load15: 0.3,
      uptime: 90061,
      diskTotal: 1000,
      diskFree: 200,
      netRxTotal: 2048,
      netTxTotal: 1024,
    };
    const wrapper = mount(HostCard, {
      props: { host: hostWithMetrics },
      global: {
        plugins: [
          i18n,
          createTestingPinia({ createSpy: vi.fn, initialState: { container: { containers: [] } } }),
        ],
        stubs: { MetricCard: MetricCardStub, HostIcon: true },
      },
    });

    const text = wrapper.text();
    expect(text).toContain("Disk");
    expect(text).toContain("Load");
    expect(text).toContain("Uptime");
    expect(text).toContain("Network");

    // Values, not just labels: an idle-looking 0.00 load must survive, 90061s
    // is "1d 1h", the disk is 800/1000 used (80.0%) and network has a direction.
    expect(text).toContain("0.50 0.40 0.30");
    expect(text).toContain("1d 1h");
    expect(text).toContain("80.0%");
    expect(text).toContain("↓");
  });

  test("hides host metric tiles when the host reports no metrics", () => {
    global.EventSource = EventSource;
    const wrapper = mount(HostCard, {
      props: { host },
      global: {
        plugins: [
          i18n,
          createTestingPinia({ createSpy: vi.fn, initialState: { container: { containers: [] } } }),
        ],
        stubs: { MetricCard: MetricCardStub, HostIcon: true },
      },
    });

    // `host` has no metricsAvailable, so nothing should render as host metrics.
    expect(wrapper.text()).not.toContain("Disk");
    expect(wrapper.text()).not.toContain("Load");
  });
});
