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
        core: "No cores | 1 core | {count} cores",
        reclaimable: "Reclaimable",
        "unused-images": "Unused images",
        "unused-volumes": "Unused volumes",
        "stopped-containers": "Stopped containers",
        "build-cache": "Build cache",
      },
      tooltip: {
        "reclaimable-source": "Same total as {command} reports.",
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

  test("renders host metrics when the host reports them", () => {
    global.EventSource = EventSource;
    const hostWithMetrics: Host = {
      ...host,
      metricsAvailable: true,
      load1: 0.5,
      load5: 0.4,
      load15: 0.3,
      uptime: 90061,
      diskTotal: 1000,
      diskFree: 200,
    };
    const wrapper = mount(HostCard, {
      props: { host: hostWithMetrics },
      global: {
        plugins: [i18n, createTestingPinia({ createSpy: vi.fn, initialState: { container: { containers: [] } } })],
        stubs: { MetricCard: MetricCardStub, HostIcon: true },
      },
    });

    const text = wrapper.text();
    expect(text).toContain("Load");
    expect(text).toContain("Uptime");
    expect(text).toContain("Disk");

    // Values, not just labels: the 1m load shows, 5m and 15m sit in the tooltip,
    // 90061s is "1d 1h", and the disk is 800/1000 used (80%).
    expect(text).toContain("0.50");
    expect(text).not.toContain("0.40");
    expect(wrapper.find('[title="1m 0.50 · 5m 0.40 · 15m 0.30 · 4 cores"]').exists()).toBe(true);
    expect(text).toContain("1d 1h");
    expect(text).toContain("80%");
    expect(text).not.toContain("Network");
  });

  test("hides host metrics when the host reports none", () => {
    global.EventSource = EventSource;
    const wrapper = mount(HostCard, {
      props: { host },
      global: {
        plugins: [i18n, createTestingPinia({ createSpy: vi.fn, initialState: { container: { containers: [] } } })],
        stubs: { MetricCard: MetricCardStub, HostIcon: true },
      },
    });

    // `host` has no metricsAvailable and no disk, so nothing should render.
    expect(wrapper.text()).not.toContain("Disk");
    expect(wrapper.text()).not.toContain("Load");
  });

  function mountWith(extra: Partial<Host>) {
    global.EventSource = EventSource;
    return mount(HostCard, {
      props: { host: { ...host, ...extra } },
      global: {
        plugins: [i18n, createTestingPinia({ createSpy: vi.fn, initialState: { container: { containers: [] } } })],
        stubs: { MetricCard: MetricCardStub, HostIcon: true },
      },
    });
  }

  // The total is `docker system df`'s RECLAIMABLE, and the tooltip says where it sits.
  // It shows on its own, without /proc metrics, since it comes from the engine.
  test("shows what the host could reclaim", () => {
    const wrapper = mountWith({
      reclaimable: {
        images: 31,
        imagesSize: 4 * 1024 ** 3,
        volumes: 2,
        volumesSize: 512 * 1024 ** 2,
        containers: 4,
        containersSize: 256 * 1024 ** 2,
        buildCacheSize: 256 * 1024 ** 2,
      },
    });

    const trigger = wrapper.findAll("button").find((b) => b.text().startsWith("Reclaimable"));
    expect(trigger?.text()).toBe("Reclaimable 5 GB");

    // the panel lists every kind with its count and size, in a fixed order
    const rows = wrapper.findAll("li").map((li) => li.findAll("span").map((s) => s.text()));
    expect(rows).toEqual([
      ["Unused images", "31", "4 GB"],
      ["Unused volumes", "2", "512 MB"],
      ["Stopped containers", "4", "256 MB"],
      ["Build cache", "256 MB"],
    ]);
    expect(wrapper.text()).toContain("Same total as docker system df reports.");
  });

  test("hides reclaimable before the first measurement", () => {
    expect(mountWith({ reclaimable: null }).text()).not.toContain("Reclaimable");
  });

  // nCPU is 4: load only takes color once there is more than one runnable task per core.
  test.each([
    [3.9, "text-base-content/80"],
    [4.1, "text-warning"],
    [8.1, "text-error"],
  ])("load %s on 4 cores is %s", (load1, expected) => {
    const wrapper = mountWith({ metricsAvailable: true, load1 });
    expect(wrapper.find(`span.font-mono.${expected.replace("/", "\\/")}`).text()).toBe(load1.toFixed(2));
  });

  // The bar follows the fullest drive, since that is the one that runs out, and the
  // tooltip names each one once there is more than Docker's.
  test("extra drives join the disk read-out", () => {
    const wrapper = mountWith({
      diskTotal: 1000,
      diskFree: 600,
      disks: [{ name: "media", total: 1000, free: 50 }],
    });

    expect(wrapper.text()).toContain("95%");
    const title = wrapper.find('[title*="media"]').attributes("title")!;
    expect(title.split("\n")).toEqual(["Docker 400 Bytes / 1000 Bytes (40%)", "media 950 Bytes / 1000 Bytes (95%)"]);
  });
});
