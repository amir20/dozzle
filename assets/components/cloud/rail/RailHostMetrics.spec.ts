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
vi.mock("@/composable/cloud/cloudConfig", () => ({
  useCloudConfig: () => ({ metricsLookbackHours: ref(24), ensureCloudStatus: vi.fn() }),
}));

const BarChartStub = defineComponent({
  name: "BarChart",
  props: ["chartData", "barClass", "max"],
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
          messages: { en: { "cloud-rail": { "metrics-failed": "Could not load the history." } } },
        }),
      ],
      stubs: { BarChart: BarChartStub },
    },
  });
  await flushPromises();
  return wrapper;
}

describe("<RailHostMetrics />", () => {
  test("draws CPU and memory per host, named the way Dozzle names it", async () => {
    respond(200, {
      bucket: 60_000,
      hosts: [
        { hostId: "h1", points: [{ ts: 1, cpu: 10, memoryUsage: 2 * 1024 ** 3 }] },
        { hostId: "gone", points: [{ ts: 1, cpu: 5, memoryUsage: 1024 }] },
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
});
