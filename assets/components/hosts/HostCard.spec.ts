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

const i18n = createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false, messages: { en: {} } });

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
});
