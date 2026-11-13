/**
 * @vitest-environment jsdom
 */
import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, describe, expect, test, vi } from "vitest";
import { computed, defineComponent, ref } from "vue";
import { createI18n } from "vue-i18n";
import RailMetrics from "./RailMetrics.vue";

// jsdom lays nothing out, so the panel would never learn its width and never
// load. 552px is the chart's content width: 552 / 5 = 110 columns.
vi.mock("@vueuse/core", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@vueuse/core")>();
  return { ...actual, useElementSize: () => ({ width: ref(552), height: ref(800) }) };
});
vi.mock("@/stores/config", () => ({ __esModule: true, default: { base: "", hosts: [] }, withBase: (p: string) => p }));
vi.mock("@/composable/logs/viewContext", () => ({
  useViewContext: () => computed(() => ({ containers: [{ id: "abc" }] })),
}));
// The deploys Dozzle itself remembers. Tests set them per case.
const updates = vi.hoisted(() => ({ list: [] as object[] }));
vi.mock("@/stores/container", async () => {
  const { reactive } = await import("vue");
  return {
    useContainerStore: () =>
      reactive({ allContainersById: { abc: { id: "abc", name: "api", host: "local" } }, recentUpdates: updates.list }),
  };
});
vi.mock("@/composable/cloud/cloudConfig", () => ({
  useCloudConfig: () => ({
    metricsLookbackHours: ref(168),
    ensureCloudStatus: vi.fn(),
    cloudConfig: ref({ linked: true }),
  }),
}));

const TimelineStub = defineComponent({
  name: "MetricsTimeline",
  props: ["from", "to", "alerts", "deploys", "format"],
  template: "<div data-timeline-stub></div>",
});

// The chart is covered by its own spec. Here it only needs to say what it was
// handed and to let a test move the shared pointer.
const BarChartStub = defineComponent({
  name: "BarChart",
  props: ["chartData", "barClass", "hoverIndex"],
  emits: ["hoverValue", "hoverEnd"],
  template: "<div data-chart></div>",
});

const HOUR = 3_600_000;

// Metrics, alerts and cloud deploys each answer from their own route.
function respond(points: object[], { hits = [] as object[], cloudDeploys = [] as object[] } = {}) {
  globalThis.fetch = vi.fn(async (input: RequestInfo | URL) => {
    const url = String(input);
    if (url.includes("/api/cloud/alerts")) return new Response(JSON.stringify({ hits }));
    if (url.includes("/api/cloud/deploys/recent")) return new Response(JSON.stringify(cloudDeploys));
    return new Response(JSON.stringify({ points, bucket: HOUR }));
  }) as typeof fetch;
}

const metricsCalls = () => vi.mocked(globalThis.fetch).mock.calls.filter(([u]) => String(u).includes("/metrics"));

async function render() {
  const wrapper = mount(RailMetrics, {
    global: {
      plugins: [
        createI18n({
          legacy: false,
          locale: "en",
          missingWarn: false,
          fallbackWarn: false,
          messages: {
            en: {
              label: { cpu: "CPU", mem: "MEM", network: "Network", "disk-io": "Disk I/O" },
              "cloud-rail": { "metrics-peak": "peak", "window-7d": "7d", "window-24h": "24h" },
            },
          },
        }),
      ],
      stubs: { BarChart: BarChartStub, MetricsTimeline: TimelineStub },
    },
  });
  await flushPromises();
  return wrapper;
}

const sections = (wrapper: Awaited<ReturnType<typeof render>>) =>
  wrapper.findAll("section").map((s) => s.text().replace(/\s+/g, " "));

afterEach(() => {
  vi.restoreAllMocks();
  updates.list.length = 0;
});

describe("<RailMetrics />", () => {
  // A fixed 120 buckets in a ~100-column chart folded pairs into 60 bars with a
  // wide gap between each.
  test("asks cloud for one bucket per chart column", async () => {
    respond([]);
    await render();

    expect(metricsCalls()[0][0]).toContain("buckets=110");
  });

  // The bug this guards: the peak was the highest bucket average, so a week of
  // hourly averages read lower than the day of 12-minute ones inside it.
  // A slow answer for the window the reader just left must not overwrite the one
  // they switched to.
  test("only the newest request's answer is shown", async () => {
    let releaseOld!: () => void;
    const body = (cpu: number) =>
      new Response(JSON.stringify({ points: [{ ts: 0, cpu, memory: 1, memoryUsage: 1 }], bucket: HOUR }));
    let metrics = 0;
    globalThis.fetch = vi.fn((input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/deploys/recent")) return Promise.resolve(new Response("[]"));
      if (!url.includes("/metrics")) return Promise.resolve(new Response(JSON.stringify({ hits: [] })));
      // The first window's answer is held back until after the second's.
      return ++metrics === 1
        ? new Promise<Response>((resolve) => (releaseOld = () => resolve(body(99))))
        : Promise.resolve(body(3));
    }) as typeof fetch;
    const wrapper = await render();
    await wrapper.findAll("[role=tab]")[0].trigger("click");
    await flushPromises();
    releaseOld();
    await flushPromises();

    expect(sections(wrapper)[0]).toContain("3.0% / peak");
  });

  test("the peak is the highest sample, not the highest average", async () => {
    respond([
      {
        ts: 0,
        cpu: 5,
        memory: 40,
        memoryUsage: 400 * 1024 ** 2,
        cpuMax: 9,
        memoryMax: 45,
        memoryUsageMax: 450 * 1024 ** 2,
      },
      {
        ts: HOUR,
        cpu: 8,
        memory: 41,
        memoryUsage: 410 * 1024 ** 2,
        cpuMax: 16,
        memoryMax: 50,
        memoryUsageMax: 500 * 1024 ** 2,
      },
    ]);
    const [cpu, mem] = sections(await render());

    expect(cpu).toContain("16.0% / peak");
    expect(mem).toContain("500M / peak");
  });

  test("an older cloud without max fields falls back to the averages", async () => {
    respond([
      { ts: 0, cpu: 5, memory: 40, memoryUsage: 400 * 1024 ** 2 },
      { ts: HOUR, cpu: 8, memory: 41, memoryUsage: 410 * 1024 ** 2 },
    ]);
    const wrapper = await render();
    const [cpu] = sections(wrapper);

    expect(cpu).toContain("8.0% / peak");
    // No rates sent: the I/O charts are hidden rather than drawn flat at zero.
    expect(wrapper.findAll("section")).toHaveLength(2);
  });

  test("network and disk show the window's totals, split in and out", async () => {
    // 1 KiB/s in and 2 KiB/s out over two one-hour buckets.
    const io = { networkRx: 1024, networkTx: 2048, diskRead: 0, diskWrite: 0 };
    respond([
      { ts: 0, cpu: 1, memory: 1, memoryUsage: 1, ...io },
      { ts: HOUR, cpu: 1, memory: 1, memoryUsage: 1, ...io },
    ]);
    const wrapper = await render();
    const [, , network, disk] = sections(wrapper);

    expect(network).toContain("Network");
    expect(network).toContain("↑14.1M ↓7M / 24h");
    // A real zero is drawn: the field was sent, so the chart stays.
    expect(disk).toContain("↑0B ↓0B");
  });

  test("hovering one chart moves the readout on every chart", async () => {
    respond([
      { ts: 0, cpu: 5, memory: 1, memoryUsage: 100 * 1024 ** 2, cpuMax: 12, networkRx: 1024, networkTx: 0 },
      { ts: HOUR, cpu: 7, memory: 1, memoryUsage: 200 * 1024 ** 2, cpuMax: 30, networkRx: 3072, networkTx: 2048 },
    ]);
    const wrapper = await render();
    wrapper.findAllComponents(BarChartStub)[2].vm.$emit("hoverValue", 0, 1, 2);
    await flushPromises();

    const [cpu, mem, network] = sections(wrapper);
    expect(cpu).toContain("7.0% · peak 30.0%");
    expect(mem).toContain("200M");
    expect(network).toContain("↑2K/s ↓3K/s");
    // Every chart is handed the same point to mark.
    for (const chart of wrapper.findAllComponents(BarChartStub)) expect(chart.props("hoverIndex")).toBe(1);
  });

  test("an I/O bar is both directions together", async () => {
    respond([{ ts: 0, cpu: 1, memory: 1, memoryUsage: 1, networkRx: 10, networkTx: 20 }]);
    const network = (await render()).findAllComponents(BarChartStub)[2];

    expect(network.props("chartData")).toEqual([{ percent: 30, value: 30 }]);
  });

  test("a level chart hands the chart its max only when cloud sent one", async () => {
    respond([{ ts: 0, cpu: 5, memory: 1, memoryUsage: 1, cpuMax: 12 }]);
    const [cpu, mem] = (await render()).findAllComponents(BarChartStub);

    expect(cpu.props("chartData")).toEqual([{ percent: 5, value: 5, peakPercent: 12, peak: 12 }]);
    expect(mem.props("chartData")).toEqual([{ percent: 1, value: 1 }]);
  });

  test("leaving the chart clears the readout everywhere", async () => {
    respond([{ ts: 0, cpu: 5, memory: 1, memoryUsage: 1, cpuMax: 12 }]);
    const wrapper = await render();
    const cpu = wrapper.findAllComponents(BarChartStub)[0];
    cpu.vm.$emit("hoverValue", 5, 0, 1);
    await flushPromises();
    cpu.vm.$emit("hoverEnd");
    await flushPromises();

    expect(sections(wrapper)[0]).toContain("12.0% / peak");
    expect(cpu.props("hoverIndex")).toBeNull();
  });

  // A deploy replaces the container id, so it is matched by host and name.
  test("the timeline gets this container's deploys and the alerts in its window", async () => {
    updates.list.push(
      { host: "local", name: "api", oldId: "o1", newId: "n1", at: new Date(HOUR / 2).toISOString() },
      { host: "local", name: "web", oldId: "o2", newId: "n2", at: new Date(HOUR / 2).toISOString() },
      { host: "other", name: "api", oldId: "o3", newId: "n3", at: new Date(HOUR / 2).toISOString() },
    );
    const hit = { alertId: "x", containerId: "abc", ts: (HOUR / 2) * 1e6, headline: "OOM", level: "error" };
    respond(
      [
        { ts: 0, cpu: 1, memory: 1, memoryUsage: 1 },
        { ts: HOUR, cpu: 1, memory: 1, memoryUsage: 1 },
      ],
      { hits: [hit] },
    );
    const wrapper = await render();
    const timeline = wrapper.findComponent(TimelineStub);

    expect(timeline.props("deploys").map((d: { newId: string }) => d.newId)).toEqual(["n1"]);
    expect(timeline.props("alerts")).toEqual([hit]);
    // The window the bars cover, first bucket's start to last bucket's end.
    expect([timeline.props("from"), timeline.props("to")]).toEqual([0, 2 * HOUR]);
    const alertsUrl = String(vi.mocked(globalThis.fetch).mock.calls.find(([u]) => String(u).includes("/alerts"))![0]);
    // The container the deploy replaced is asked for too: its alerts are the
    // ones that explain the step.
    expect(alertsUrl).toContain("containerIds=abc%2Co1");
    expect(alertsUrl).toContain(`to=${2 * HOUR * 1e6}`);
  });

  test("hovering a bar names what happened in it", async () => {
    const hit = { alertId: "x", containerId: "abc", ts: (HOUR + 60_000) * 1e6, headline: "OOM killed", level: "error" };
    respond(
      [
        { ts: 0, cpu: 1, memory: 1, memoryUsage: 1 },
        { ts: HOUR, cpu: 1, memory: 1, memoryUsage: 1 },
      ],
      { hits: [hit] },
    );
    const wrapper = await render();
    const chart = wrapper.findAllComponents(BarChartStub)[0];

    chart.vm.$emit("hoverValue", 1, 0, 2);
    await flushPromises();
    expect(wrapper.text()).not.toContain("OOM killed");

    chart.vm.$emit("hoverValue", 1, 1, 2);
    await flushPromises();
    expect(wrapper.text()).toContain("OOM killed");
  });

  // After a restart only Cloud remembers the deploy, and its answer can land
  // after the metrics. The container it replaced must still be asked for.
  test("waits for Cloud's deploys before asking for alerts", async () => {
    let releaseDeploys!: () => void;
    const cloudDeploy = {
      host: "local",
      name: "api",
      oldId: "old1",
      newId: "abc",
      at: new Date(HOUR / 2).toISOString(),
    };
    globalThis.fetch = vi.fn((input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/deploys/recent"))
        return new Promise<Response>(
          (resolve) => (releaseDeploys = () => resolve(new Response(JSON.stringify([cloudDeploy])))),
        );
      if (url.includes("/alerts")) return Promise.resolve(new Response(JSON.stringify({ hits: [] })));
      return Promise.resolve(
        new Response(
          JSON.stringify({
            points: [
              { ts: 0, cpu: 1, memory: 1, memoryUsage: 1 },
              { ts: HOUR, cpu: 1, memory: 1, memoryUsage: 1 },
            ],
            bucket: HOUR,
          }),
        ),
      );
    }) as typeof fetch;
    await render();
    const alertCalls = () => vi.mocked(globalThis.fetch).mock.calls.filter(([u]) => String(u).includes("/alerts"));
    expect(alertCalls()).toHaveLength(0);

    releaseDeploys();
    await flushPromises();
    expect(String(alertCalls()[0][0])).toContain("containerIds=abc%2Cold1");
  });
});
