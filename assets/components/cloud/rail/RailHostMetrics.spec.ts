/**
 * @vitest-environment jsdom
 */
import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { defineComponent, ref } from "vue";
import { createI18n } from "vue-i18n";
import RailHostMetrics from "./RailHostMetrics.vue";

// jsdom lays nothing out; 552 / 5 = 110 columns.
vi.mock("@vueuse/core", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@vueuse/core")>();
  return { ...actual, useElementSize: () => ({ width: ref(552), height: ref(800) }) };
});
vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [{ id: "h1", name: "orbstack", memTotal: 8 * 1024 ** 3, available: true }] },
  withBase: (p: string) => p,
}));
// The live samples the panel falls back on, one running container on h1.
const live = vi.hoisted(() => ({
  containers: [] as object[],
}));
vi.mock("@/stores/container", () => ({ useContainerStore: () => ({ containers: live.containers }) }));
vi.mock("@/composable/cloud/cloudConfig", () => ({
  useCloudConfig: () => ({ metricsLookbackHours: ref(24), ensureCloudStatus: vi.fn() }),
}));

const BarChartStub = defineComponent({
  name: "BarChart",
  props: ["chartData", "barClass", "max", "hoverIndex", "sampledFrom"],
  emits: ["hoverValue", "hoverEnd"],
  template: "<div data-chart></div>",
});

function respond(status: number, body: object = {}) {
  globalThis.fetch = vi.fn(async () => new Response(JSON.stringify(body), { status })) as typeof fetch;
}

async function render() {
  const wrapper = mount(RailHostMetrics, {
    global: {
      plugins: [
        createI18n({
          legacy: false,
          locale: "en",
          missingWarn: false,
          fallbackWarn: false,
          messages: {
            en: {
              "cloud-rail": { "metrics-failed": "Could not load the history.", "metrics-peak": "peak" },
              analytics: { avg: "Avg" },
            },
          },
        }),
      ],
      stubs: { BarChart: BarChartStub },
    },
  });
  await flushPromises();
  return wrapper;
}

const HOUR = 3_600_000;
const GB = 1024 ** 3;
const text = (el: { text(): string }) => el.text().replace(/\s+/g, " ");
// The panel's own clock for a 24h window: the date comes along with the time.
const dayClock = new Intl.DateTimeFormat(undefined, {
  month: "short",
  day: "numeric",
  hour: "2-digit",
  minute: "2-digit",
});

describe("<RailHostMetrics />", () => {
  test("draws CPU and memory per host, named the way Dozzle names it", async () => {
    respond(200, {
      bucket: 60_000,
      hosts: [
        {
          hostId: "h1",
          points: [
            { ts: 1, cpu: 10, memoryUsage: 2 * 1024 ** 3 },
            { ts: 2, cpu: 10, memoryUsage: 2 * 1024 ** 3 },
          ],
        },
        {
          hostId: "gone",
          points: [
            { ts: 1, cpu: 5, memoryUsage: 1024 },
            { ts: 2, cpu: 5, memoryUsage: 1024 },
          ],
        },
      ],
    });
    const wrapper = await render();

    expect(vi.mocked(globalThis.fetch).mock.calls[0][0]).toBe("/api/cloud/metrics/hosts?window=24h&buckets=110");
    const sections = wrapper.findAll("[data-host]");
    expect(sections.map((s) => s.find("h3").text())).toEqual(["orbstack", "gone"]);

    // Memory reads against the host's memory where Dozzle knows it: 2 of 8 GB.
    const memory = sections[0].findAllComponents(BarChartStub)[1];
    expect(memory.props("max")).toBe(100);
    expect(memory.props("chartData")[0].percent).toBe(25);
    // A host Dozzle no longer connects to has no total to read against.
    expect(sections[1].findAllComponents(BarChartStub)[1].props("max")).toBeUndefined();
  });

  test("a failed read says so instead of drawing nothing", async () => {
    respond(502);
    const wrapper = await render();

    expect(wrapper.text()).toContain("Could not load");
    expect(wrapper.find("[data-host]").exists()).toBe(false);
  });

  // Local Cloud sends no host series yet: each host's live totals stand in,
  // summed the way its home page card sums them, and say where they came from.
  test("with no host series from Cloud, the live totals stand in", async () => {
    live.containers = [
      {
        id: "a",
        host: "h1",
        state: "running",
        statsHistory: [{ cpu: 20, memory: 1, memoryUsage: 1024 ** 3 }],
        sampledStats: 1,
      },
      {
        id: "b",
        host: "h1",
        state: "exited",
        statsHistory: [{ cpu: 50, memory: 1, memoryUsage: 1024 ** 3 }],
        sampledStats: 1,
      },
    ];
    respond(200, { bucket: 60_000, hosts: [] });
    const wrapper = await render();

    const series = wrapper.findAll("[data-series]");
    expect(series.map((s) => s.find("h3").text())).toEqual(["orbstack"]);
    const [cpu, memory] = series[0].findAllComponents(BarChartStub);
    // One running container, so the newest sample is its own: 20% of a core-less host, 1 of 8 GB.
    expect(cpu.props("chartData").at(-1).value).toBe(20);
    expect(memory.props("chartData").at(-1).percent).toBe(12.5);
    expect(wrapper.find("[data-live-note]").exists()).toBe(true);
    live.containers = [];
  });

  test("hovering a bar reads that bucket and names its time, on both of the host's charts", async () => {
    respond(200, {
      bucket: HOUR,
      hosts: [
        {
          hostId: "h1",
          points: [
            { ts: 0, cpu: 10, memoryUsage: 1 * GB },
            { ts: HOUR, cpu: 40, memoryUsage: 3 * GB },
          ],
        },
        {
          hostId: "other",
          points: [
            { ts: 0, cpu: 5, memoryUsage: 1024 },
            { ts: HOUR, cpu: 5, memoryUsage: 1024 },
          ],
        },
      ],
    });
    const wrapper = await render();
    const [first, second] = wrapper.findAll("[data-host]");

    // At rest: each chart's peak, and the window's span under the charts.
    expect(text(first)).toContain("40.0% / peak");
    expect(first.findAll("[data-range] span").map((s) => s.text())).toEqual([
      dayClock.format(new Date(0)),
      dayClock.format(new Date(2 * HOUR)),
    ]);

    first.findAllComponents(BarChartStub)[1].vm.$emit("hoverValue", 3 * GB, 0, 2);
    await flushPromises();

    const [cpu, memory] = first.findAllComponents(BarChartStub);
    expect([cpu.props("hoverIndex"), memory.props("hoverIndex")]).toEqual([0, 0]);
    expect(text(first)).toContain("10.0%");
    expect(text(first)).not.toContain("/ peak");
    expect(text(first)).toContain("1G");
    expect(first.find("[data-moment]").text()).toContain(dayClock.format(new Date(0)));
    // Another host's bars are a different series: its readout stays at rest.
    expect(second.findComponent(BarChartStub).props("hoverIndex")).toBeNull();
    expect(text(second)).toContain("5.0% / peak");

    cpu.vm.$emit("hoverEnd");
    await flushPromises();
    expect(text(first)).toContain("40.0% / peak");
    expect(cpu.props("hoverIndex")).toBeNull();
    expect(first.find("[data-moment]").text()).not.toContain(dayClock.format(new Date(0)));
  });

  test("the live fallback reads a hovered bar and names its time too", async () => {
    live.containers = [
      {
        id: "a",
        host: "h1",
        state: "running",
        statsHistory: [
          { cpu: 10, memory: 1, memoryUsage: GB },
          { cpu: 30, memory: 1, memoryUsage: 2 * GB },
        ],
        sampledStats: 2,
      },
    ];
    respond(200, { bucket: 60_000, hosts: [] });
    const wrapper = await render();
    const series = wrapper.find("[data-series]");
    const [cpu, memory] = series.findAllComponents(BarChartStub);
    // 300 samples, the last two real: bar 149 of 150 is samples 298-299.
    cpu.vm.$emit("hoverValue", 20, 149, 150);
    await flushPromises();

    expect(memory.props("hoverIndex")).toBe(149);
    expect(text(series)).toContain("20.0%");
    expect(text(series)).toContain("1.5G");
    expect(series.find("[data-moment]").exists()).toBe(true);
    expect(series.find("[data-range]").exists()).toBe(true);

    memory.vm.$emit("hoverEnd");
    await flushPromises();
    expect(text(series)).toContain("30.0% / peak");
    expect(series.find("[data-moment]").exists()).toBe(false);
    live.containers = [];
  });
});
