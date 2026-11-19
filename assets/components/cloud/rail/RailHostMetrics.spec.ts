/**
 * @vitest-environment jsdom
 */
import { flushPromises, mount, RouterLinkStub } from "@vue/test-utils";
import { afterEach, describe, expect, test, vi } from "vitest";
import { computed, defineComponent, ref } from "vue";
import { createI18n } from "vue-i18n";
import RailHostMetrics from "./RailHostMetrics.vue";

// jsdom lays nothing out; 552 / 5 = 110 columns.
vi.mock("@vueuse/core", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@vueuse/core")>();
  return { ...actual, useElementSize: () => ({ width: ref(552), height: ref(800) }) };
});
const state = vi.hoisted(() => ({
  hosts: {} as Record<string, object>,
  containers: [] as object[],
  inView: [] as { id: string }[],
}));
vi.mock("@/stores/hosts", () => ({ useHosts: () => ({ hosts: computed(() => state.hosts) }) }));
vi.mock("@/stores/container", () => ({
  useContainerStore: () => ({
    containers: state.containers,
    allContainersById: Object.fromEntries((state.containers as { id: string }[]).map((c) => [c.id, c])),
  }),
}));
vi.mock("@/composable/logs/viewContext", () => ({
  useViewContext: () => computed(() => ({ containers: state.inView })),
}));
vi.mock("@/composable/cloud/cloudRail", () => ({ useCloudRail: () => ({ closeRail: vi.fn() }) }));
vi.mock("@/composable/cloud/cloudConfig", () => ({
  useCloudConfig: () => ({ metricsLookbackHours: ref(24), ensureCloudStatus: vi.fn() }),
}));
vi.mock("@/stores/config", () => ({ __esModule: true, default: { base: "" }, withBase: (p: string) => p }));

const BarChartStub = defineComponent({
  name: "BarChart",
  props: ["chartData", "barClass", "max", "hoverIndex"],
  emits: ["hoverValue", "hoverEnd"],
  template: "<div></div>",
});
// The menu's content renders inline, so a test can read it without opening it.
const PopoverStub = defineComponent({
  name: "Popover",
  template: '<div><slot name="trigger" /><slot :close="() => {}" /></div>',
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
              "cloud-rail": {
                "metrics-failed": "Could not load the history.",
                "metrics-hosts-empty": "No host history in Cloud yet.",
                "metrics-peak": "peak",
                "scope-all-hosts": "All hosts",
                "metrics-cpu-avg-host": "CPU · avg host",
                "metrics-containers-summed": "{n} containers summed",
                "metrics-whole-host": "Whole host",
                "window-1h": "1h",
              },
              label: { cpu: "CPU", mem: "MEM", network: "Network", "disk-io": "Disk I/O" },
              analytics: { avg: "Avg" },
            },
          },
        }),
      ],
      stubs: { BarChart: BarChartStub, Popover: PopoverStub, RouterLink: RouterLinkStub },
    },
  });
  await flushPromises();
  return wrapper;
}

const GB = 1024 ** 3;
const text = (el: { text(): string }) => el.text().replace(/\s+/g, " ");
const top = (cpu: { name: string; avg: number; peak: number }[]) => ({ cpu, memory: [], network: [], disk: [] });
const series = (hostId: string, cpu: [number, number], memoryUsage: number, extra: object = {}) => ({
  hostId,
  containers: 4,
  points: [
    { ts: 0, cpu: cpu[0], memoryUsage, ...extra },
    { ts: 60_000, cpu: cpu[1], memoryUsage, ...extra },
  ],
});
const charts = (wrapper: Awaited<ReturnType<typeof render>>) =>
  Object.fromEntries(wrapper.findAll("[data-chart]").map((s) => [s.attributes("data-chart"), s]));

afterEach(() => {
  state.hosts = {};
  state.containers = [];
  state.inView = [];
});

describe("<RailHostMetrics />", () => {
  test("opens on every host combined: CPU is the average host, memory and I/O add up", async () => {
    state.hosts = { h1: { name: "orbstack", type: "local" }, h2: { name: "nas", type: "agent" } };
    respond(200, {
      bucket: 60_000,
      hosts: [
        series("h1", [10, 30], 2 * GB, { networkRx: 100, networkTx: 50 }),
        series("h2", [50, 70], 1 * GB, { networkRx: 300, networkTx: 0 }),
      ],
    });
    const wrapper = await render();

    expect(vi.mocked(globalThis.fetch).mock.calls[0][0]).toBe("/api/cloud/metrics/hosts?window=24h&buckets=110");
    expect(text(wrapper.find("[data-subject]"))).toContain("All hosts");
    expect(text(wrapper.find("[data-subject]"))).toContain("8 containers summed");
    const { cpu, memory, network, disk } = charts(wrapper);
    expect(cpu.text()).toContain("CPU · avg host");
    expect(
      cpu
        .findComponent(BarChartStub)
        .props("chartData")
        .map((d: { value: number }) => d.value),
    ).toEqual([30, 50]);
    expect(memory.findComponent(BarChartStub).props("chartData")[0].value).toBe(3 * GB);
    expect(network.findComponent(BarChartStub).props("chartData")[0].value).toBe(450);
    // No disk rates from Cloud: hidden rather than drawn as idle.
    expect(disk).toBeUndefined();
    // A few hosts are one segment each.
    expect(wrapper.findAll("[data-host-segments] button").map((b) => b.text())).toEqual([
      "All hosts",
      "orbstack",
      "nas",
    ]);
  });

  test("a host row narrows every chart to that host, and the header says it is the whole host", async () => {
    state.hosts = { h1: { name: "orbstack", type: "local" }, h2: { name: "nas", type: "agent" } };
    respond(200, { bucket: 60_000, hosts: [series("h1", [10, 30], 2 * GB), series("h2", [50, 70], 1 * GB)] });
    const wrapper = await render();

    // Ranked busiest first.
    const rows = wrapper.findAll("[data-host-ranking] button");
    expect(rows.map((r) => r.find("span").text())).toEqual(["nas", "orbstack"]);
    await rows[1].trigger("click");

    const subject = text(wrapper.find("[data-subject]"));
    expect(subject).toContain("orbstack");
    expect(subject).toContain("Whole host · 4 containers summed");
    expect(wrapper.find("[data-host-ranking]").exists()).toBe(false);
    const { cpu } = charts(wrapper);
    expect(cpu.text()).not.toContain("avg host");
    expect(
      cpu
        .findComponent(BarChartStub)
        .props("chartData")
        .map((d: { value: number }) => d.value),
    ).toEqual([10, 30]);
  });

  test("a peak comes from Cloud's max, not the highest average", async () => {
    state.hosts = { h1: { name: "orbstack" } };
    respond(200, { bucket: 60_000, hosts: [series("h1", [10, 30], GB, { cpuMax: 95 })] });
    const wrapper = await render();

    expect(text(charts(wrapper).cpu)).toContain("95.0% / peak");
  });

  test("on a container page it opens on that container's host", async () => {
    state.hosts = { h1: { name: "orbstack" }, h2: { name: "nas" } };
    state.containers = [{ id: "c1", host: "h2", name: "plex" }];
    state.inView = [{ id: "c1" }];
    respond(200, { bucket: 60_000, hosts: [series("h1", [10, 30], GB), series("h2", [50, 70], GB)] });
    const wrapper = await render();

    expect(text(wrapper.find("[data-subject]"))).toContain("nas");
  });

  test("past four hosts the picker is a menu with a filter, busiest first", async () => {
    const ids = ["a", "b", "c", "d", "e"];
    state.hosts = Object.fromEntries(ids.map((id) => [id, { name: `host-${id}` }]));
    respond(200, {
      bucket: 60_000,
      hosts: ids.map((id, i) => series(id, [i * 10, i * 10], GB)),
    });
    const wrapper = await render();

    expect(wrapper.find("[data-host-segments]").exists()).toBe(false);
    const menu = wrapper.find("[data-host-menu]");
    const names = () => menu.findAll("[data-host-option] span.truncate").map((s) => s.text());
    expect(names()).toEqual(["host-e", "host-d", "host-c", "host-b", "host-a"]);
    await menu.find("input").setValue("b");
    expect(names()).toEqual(["host-b"]);
  });

  test("top containers merge across hosts, name their host, and link to the container", async () => {
    state.hosts = { h1: { name: "orbstack" }, h2: { name: "nas" } };
    state.containers = [{ id: "c9", host: "h2", name: "plex" }];
    respond(200, {
      bucket: 60_000,
      hosts: [
        { ...series("h1", [10, 30], GB), top: top([{ name: "api", avg: 5, peak: 9 }]) },
        { ...series("h2", [50, 70], GB), top: top([{ name: "plex", avg: 40, peak: 80 }]) },
      ],
    });
    const wrapper = await render();

    const rows = wrapper.find("[data-top]").findAll(":scope > *");
    expect(rows.map((r) => text(r.find("span.truncate")))).toEqual(["plex · nas", "api · orbstack"]);
    expect(rows.map((r) => text(r.find("span.whitespace-nowrap")))).toEqual(["40.0% · 80.0%", "5.0% · 9.0%"]);
    // plex still exists on that host; api is not known to Dozzle, so it is not a link.
    expect(wrapper.findComponent(RouterLinkStub).props("to")).toEqual({
      name: "/container/[id]",
      params: { id: "c9" },
    });
    expect(wrapper.findAllComponents(RouterLinkStub)).toHaveLength(1);
  });

  // The server drops the lists for a reader with a container filter.
  test("without the lists there is no top containers section", async () => {
    state.hosts = { h1: { name: "orbstack" } };
    respond(200, { bucket: 60_000, hosts: [series("h1", [10, 30], GB)] });
    const wrapper = await render();

    expect(wrapper.find("[data-top]").exists()).toBe(false);
  });

  test("a failed read says so, and an empty one says Cloud has no history yet", async () => {
    respond(502);
    expect((await render()).text()).toContain("Could not load");

    respond(200, { bucket: 60_000, hosts: [] });
    const wrapper = await render();
    expect(wrapper.text()).toContain("No host history in Cloud yet.");
    expect(wrapper.find("[data-chart]").exists()).toBe(false);
  });

  test("hovering a bar reads that bucket on every chart and names its time", async () => {
    state.hosts = { h1: { name: "orbstack" } };
    respond(200, { bucket: 60_000, hosts: [series("h1", [10, 30], GB)] });
    const wrapper = await render();
    const { cpu, memory } = charts(wrapper);

    memory.findComponent(BarChartStub).vm.$emit("hoverValue", GB, 0, 2);
    await flushPromises();

    expect(cpu.findComponent(BarChartStub).props("hoverIndex")).toBe(0);
    expect(text(charts(wrapper).cpu)).toContain("10.0%");
    expect(text(charts(wrapper).cpu)).not.toContain("/ peak");
    expect(wrapper.find("[data-moment]").text()).toContain("·");
  });

  // An older cloud sends no rates: ranking by them would read as zero traffic.
  test("Network and Disk are not offered as rankings when Cloud sends no rates", async () => {
    state.hosts = { h1: { name: "orbstack" }, h2: { name: "nas" } };
    respond(200, { bucket: 60_000, hosts: [series("h1", [10, 30], GB), series("h2", [50, 70], GB)] });
    const wrapper = await render();

    expect(wrapper.find("[data-host-ranking]").exists()).toBe(true);
    const tabs = wrapper
      .findAll('[role="tablist"]')
      .at(-1)!
      .findAll("button")
      .map((b) => b.text());
    expect(tabs).toEqual(["CPU", "MEM"]);
  });

  // Picking "All hosts" on a container page sticks, even when a new window
  // changes which hosts answered.
  test("a reader's pick survives a window change on a container page", async () => {
    state.hosts = { h1: { name: "orbstack" }, h2: { name: "nas" } };
    state.containers = [{ id: "c1", host: "h2", name: "plex" }];
    state.inView = [{ id: "c1" }];
    respond(200, { bucket: 60_000, hosts: [series("h1", [10, 30], GB), series("h2", [50, 70], GB)] });
    const wrapper = await render();
    expect(text(wrapper.find("[data-subject]"))).toContain("nas");

    await wrapper.find("[data-subject] button").trigger("click");
    // A single host's header says "Whole host"; the combined one does not.
    expect(text(wrapper.find("[data-subject]"))).not.toContain("Whole host");

    respond(200, {
      bucket: 60_000,
      hosts: [series("h1", [10, 30], GB), series("h2", [50, 70], GB), series("h3", [1, 2], GB)],
    });
    await wrapper
      .findAll('[role="tab"]')
      .find((t) => t.text() === "1h")!
      .trigger("click");
    await flushPromises();
    expect(text(wrapper.find("[data-subject]"))).not.toContain("Whole host");
    expect(text(wrapper.find("[data-subject]"))).toContain("12 containers summed");
  });
});
