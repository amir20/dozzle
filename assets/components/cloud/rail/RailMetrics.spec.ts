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
vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [{ id: "local", name: "local", memTotal: 32 * 1024 ** 3 }] },
  withBase: (p: string) => p,
}));
vi.mock("@/composable/logs/viewContext", () => ({
  useViewContext: () => computed(() => ({ containers: [{ id: "abc" }] })),
}));
// The deploys Dozzle itself remembers. Tests set them per case.
const updates = vi.hoisted(() => ({
  list: [] as object[],
  created: 0,
  memoryLimit: 0,
  // The container's live samples, for the fallback while Cloud has too little.
  live: [] as { cpu: number; memory: number; memoryUsage: number }[],
}));
const jump = vi.hoisted(() => ({ jumpTo: vi.fn() }));
vi.mock("@/composable/logs/logJump", () => ({ useLogJump: () => jump }));
const rail = vi.hoisted(() => ({ hover: true, closeRail: vi.fn() }));
// A getter rather than a computed: this module is evaluated once, and a computed
// over a plain object would cache its first answer for every test.
vi.mock("@/composable/ui/media", () => ({
  canHover: {
    get value() {
      return rail.hover;
    },
  },
  isMobile: { value: false },
  isMac: true,
}));
vi.mock("@/composable/cloud/cloudRail", () => ({ useCloudRail: () => ({ closeRail: rail.closeRail }) }));
vi.mock("@/stores/container", async () => {
  const { reactive } = await import("vue");
  return {
    useContainerStore: () =>
      reactive({
        allContainersById: {
          abc: {
            id: "abc",
            name: "api",
            host: "local",
            created: new Date(updates.created),
            memoryLimit: updates.memoryLimit,
            cpuLimit: 0,
            statsHistory: updates.live,
            sampledStats: updates.live.length,
          },
        },
        recentUpdates: updates.list,
      }),
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
  props: ["chartData", "barClass", "hoverIndex", "max"],
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
              analytics: { avg: "Avg", max: "Max" },
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
  updates.created = 0;
  updates.memoryLimit = 0;
  updates.live = [];
  jump.jumpTo.mockReset();
  rail.closeRail.mockReset();
  rail.hover = true;
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
      new Response(
        JSON.stringify({
          points: [
            { ts: 0, cpu, memory: 1, memoryUsage: 1 },
            { ts: HOUR, cpu, memory: 1, memoryUsage: 1 },
          ],
          bucket: HOUR,
        }),
      );
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
    const point = { cpu: 1, memory: 1, memoryUsage: 1, networkRx: 10, networkTx: 20 };
    respond([
      { ts: 0, ...point },
      { ts: HOUR, ...point },
    ]);
    const network = (await render()).findAllComponents(BarChartStub)[2];

    expect(network.props("chartData")[0]).toEqual({ percent: 30, value: 30 });
  });

  test("a level chart hands the chart its max only when cloud sent one", async () => {
    const point = { cpu: 5, memory: 1, memoryUsage: 1, cpuMax: 12 };
    respond([
      { ts: 0, ...point },
      { ts: HOUR, ...point },
    ]);
    const [cpu, mem] = (await render()).findAllComponents(BarChartStub);

    expect(cpu.props("chartData")[0]).toEqual({ percent: 5, value: 5, peakPercent: 12, peak: 12 });
    expect(mem.props("chartData")[0]).toEqual({ percent: 1, value: 1 });
  });

  test("leaving the chart clears the readout everywhere", async () => {
    const point = { cpu: 5, memory: 1, memoryUsage: 1, cpuMax: 12 };
    respond([
      { ts: 0, ...point },
      { ts: HOUR, ...point },
    ]);
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

  // A bar is a moment; clicking it opens this container's logs there.
  test("clicking a bar jumps to the logs at its moment", async () => {
    respond([
      { ts: 0, cpu: 1, memory: 1, memoryUsage: 1 },
      { ts: HOUR, cpu: 1, memory: 1, memoryUsage: 1 },
    ]);
    const wrapper = await render();
    wrapper.findAllComponents(BarChartStub)[0].vm.$emit("hoverValue", 1, 1, 2);
    await flushPromises();
    await wrapper.findAll("section button")[0].trigger("click");

    expect(jump.jumpTo).toHaveBeenCalledWith({ containerId: "abc", date: new Date(HOUR) });
    // The drawer is modal over the logs, so it closes or the jump is unseen.
    expect(rail.closeRail).toHaveBeenCalled();
  });

  // Before this container was created the bars are one a deploy replaced, and
  // its logs went with it.
  test("a bar from before the container existed does not jump", async () => {
    updates.created = HOUR;
    respond([
      { ts: 0, cpu: 1, memory: 1, memoryUsage: 1 },
      { ts: HOUR, cpu: 1, memory: 1, memoryUsage: 1 },
    ]);
    const wrapper = await render();
    wrapper.findAllComponents(BarChartStub)[0].vm.$emit("hoverValue", 1, 0, 2);
    await flushPromises();
    const button = wrapper.findAll("section button")[0];
    await button.trigger("click");

    expect(jump.jumpTo).not.toHaveBeenCalled();
    expect(button.attributes("aria-disabled")).toBe("true");
  });

  // An iPad in landscape is wider than a phone but has no hover either: a tap
  // there is still how a bar is read.
  test("on any touch screen a tap reads the bar and does not jump", async () => {
    rail.hover = false;
    respond([
      { ts: 0, cpu: 1, memory: 1, memoryUsage: 1 },
      { ts: HOUR, cpu: 1, memory: 1, memoryUsage: 1 },
    ]);
    const wrapper = await render();
    wrapper.findAllComponents(BarChartStub)[0].vm.$emit("hoverValue", 1, 1, 2);
    await flushPromises();
    await wrapper.findAll("section button")[0].trigger("click");

    expect(jump.jumpTo).not.toHaveBeenCalled();
  });

  // Scaled to its own peak a steady container drew as a solid block.
  test("memory is drawn against the container's limit", async () => {
    const M = 1024 ** 2;
    updates.memoryLimit = 2048 * M;
    respond([
      { ts: 0, cpu: 1, memory: 20, memoryUsage: 400 * M, memoryUsageMax: 500 * M },
      { ts: HOUR, cpu: 1, memory: 20, memoryUsage: 400 * M, memoryUsageMax: 500 * M },
    ]);
    const wrapper = await render();
    const mem = wrapper.findAllComponents(BarChartStub)[1];

    expect(mem.props("max")).toBe(100);
    expect(mem.props("chartData")[0]).toEqual({ percent: 20, value: 400 * M, peakPercent: 25, peak: 500 * M });
    expect(wrapper.findAll("[data-summary]")[1].text().replace(/\s+/g, " ")).toBe("Avg 400M / 2G");
  });

  // Docker measures an unlimited container's percent against the host's RAM;
  // drawn against that, 300M of a 32G host is a flat line.
  test("without a limit, memory keeps its own scale even when cloud sent a percent", async () => {
    respond([
      { ts: 0, cpu: 1, memory: 0.9, memoryUsage: 300 },
      { ts: HOUR, cpu: 1, memory: 0.9, memoryUsage: 300 },
    ]);
    const wrapper = await render();
    const mem = wrapper.findAllComponents(BarChartStub)[1];

    expect(mem.props("max")).toBeUndefined();
    expect(mem.props("chartData")[0]).toEqual({ percent: 300, value: 300 });
    expect(wrapper.findAll("[data-summary]")[1].text()).not.toContain("/");
  });

  // Each point's percent was measured against the limit in force then, so a
  // deploy that lowered the limit does not push the bars before it to the top.
  test("bars before a limit change keep the limit they were measured against", async () => {
    const M = 1024 ** 2;
    updates.memoryLimit = 512 * M;
    respond([
      { ts: 0, cpu: 1, memory: 25, memoryUsage: 400 * M },
      { ts: HOUR, cpu: 1, memory: 50, memoryUsage: 256 * M },
    ]);
    const mem = (await render()).findAllComponents(BarChartStub)[1];

    expect(mem.props("chartData").map((d: { percent: number }) => d.percent)).toEqual([25, 50]);
  });

  // Before the limit existed the percent was measured against the host's RAM,
  // which under a "/ 2G" header is the flat line the limit was meant to fix.
  test("bars from before a limit was added are drawn against the limit", async () => {
    const M = 1024 ** 2;
    updates.memoryLimit = 2048 * M;
    respond([
      { ts: 0, cpu: 1, memory: (300 / (32 * 1024)) * 100, memoryUsage: 300 * M, memoryUsageMax: 600 * M },
      { ts: HOUR, cpu: 1, memory: 20, memoryUsage: 400 * M },
    ]);
    const mem = (await render()).findAllComponents(BarChartStub)[1];
    const [before, after] = mem.props("chartData");

    expect(before.percent).toBeCloseTo((300 / 2048) * 100);
    expect(before.peakPercent).toBeCloseTo((600 / 2048) * 100);
    expect(after.percent).toBe(20);
  });

  test("each chart sums up its window, and the window's ends are labelled", async () => {
    respond([
      { ts: 0, cpu: 4, cpuMax: 9, memory: 1, memoryUsage: 1, networkRx: 1024, networkTx: 2048 },
      { ts: HOUR, cpu: 8, cpuMax: 20, memory: 1, memoryUsage: 1, networkRx: 3072, networkTx: 2048 },
    ]);
    const wrapper = await render();
    const summaries = wrapper.findAll("[data-summary]").map((p) => p.text());

    expect(summaries[0]).toBe("Avg 6.0%");
    expect(summaries[2]).toBe("Avg ↑2K/s ↓2K/s");
    const ends = wrapper.findAll("[data-range] span").map((s) => s.text());
    expect(ends).toHaveLength(2);
    expect(ends[0]).not.toBe(ends[1]);
  });

  // Local Cloud often has a point or none for a fresh container; the panel
  // draws what this browser already holds rather than nothing.
  test("with fewer than two points from Cloud, the live window stands in", async () => {
    updates.live = [
      { cpu: 10, memory: 1, memoryUsage: 100 },
      { cpu: 30, memory: 1, memoryUsage: 300 },
    ];
    respond([{ ts: 0, cpu: 99, memory: 1, memoryUsage: 1 }]);
    const wrapper = await render();

    expect(wrapper.find("[data-live]").exists()).toBe(true);
    const [cpu, mem] = wrapper.findAllComponents(BarChartStub);
    expect(cpu.props("chartData").map((d: { value: number }) => d.value)).toEqual([10, 30]);
    expect(mem.props("chartData").map((d: { value: number }) => d.value)).toEqual([100, 300]);
    expect(wrapper.find("[data-live-note]").exists()).toBe(true);
    expect(wrapper.find("[data-timeline-stub]").exists()).toBe(false);
  });

  test("Cloud's history replaces the live window once it has two points", async () => {
    updates.live = [{ cpu: 10, memory: 1, memoryUsage: 100 }];
    respond([
      { ts: 0, cpu: 1, memory: 1, memoryUsage: 1 },
      { ts: HOUR, cpu: 2, memory: 1, memoryUsage: 1 },
    ]);
    const wrapper = await render();

    expect(wrapper.find("[data-live]").exists()).toBe(false);
    expect(wrapper.find("[data-timeline-stub]").exists()).toBe(true);
  });
});
