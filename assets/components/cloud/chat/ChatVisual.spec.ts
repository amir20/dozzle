/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { defineComponent } from "vue";
import { createI18n } from "vue-i18n";
import type { ChatVisual as Visual } from "@/composable/cloud/chatVisuals";
import ChatVisual from "./ChatVisual.vue";

vi.mock("@/modules/i18n", () => ({
  i18n: {
    global: { t: (key: string, params?: Record<string, unknown>) => `${key}${params ? JSON.stringify(params) : ""}` },
  },
}));

// The charts are their own components with their own specs; here they only have
// to receive what the picture carried.
const BarChartStub = defineComponent({ props: ["chartData", "max"], template: "<div data-bars />" });
const VolumeStub = defineComponent({ props: ["buckets", "stepNs", "end"], template: "<div data-volume />" });

const GiB = 1024 ** 3;

function render(visual: Visual) {
  return mount(ChatVisual, {
    props: { visual },
    global: {
      plugins: [
        createI18n({
          legacy: false,
          locale: "en",
          missingWarn: false,
          fallbackWarn: false,
          messages: {
            en: {
              label: { cpu: "CPU" },
              "cloud-chat": {
                visual: {
                  memory: "Memory",
                  peak: "peak {peak}",
                  "peak-of-limit": "peak {peak} of {limit}",
                  "no-events": "No exits or restarts recorded",
                  "no-events-stored": "No events are stored for this container",
                  floor: "Older events were cut off",
                  matches: "matching lines: {n}",
                  "top-restarts": "Top containers by restarts",
                },
              },
            },
          },
        }),
      ],
      stubs: { BarChart: BarChartStub, SearchVolume: VolumeStub, LogDate: true, LogLevel: true },
    },
  });
}

describe("ChatVisual", () => {
  test("memory against the limit: peak of limit, facts line and bars drawn against 100%", () => {
    const wrapper = render({
      type: "memory_vs_limit",
      container: "rook",
      hostName: "nas",
      from: 0,
      to: 1,
      facts: {
        exits: 24,
        exitCodes: [
          { code: "137", count: 17 },
          { code: "1", count: 7 },
        ],
        ooms: 0,
        restarts: 0,
        eventsRecorded: true,
        capped: false,
        memoryPeak: 16 * GiB,
      },
      points: [{ ts: 0, memory: 0, memoryUsage: 8 * GiB }],
      memoryLimit: 16 * GiB,
    });
    expect(wrapper.text()).toContain("peak 16 GB of 16 GB");
    expect(wrapper.text()).toContain('cloud-chat.visual.exits{"n":24}');
    const bars = wrapper.findComponent(BarChartStub);
    expect(bars.props("max")).toBe(100);
    expect(bars.props("chartData")[0].percent).toBe(50);
  });

  test("CPU over time: its peak, and bars scaled to their own peak", () => {
    const wrapper = render({
      type: "cpu_over_time",
      container: "dozzle_dozzle.1",
      from: 0,
      to: 1,
      points: [
        { ts: 0, cpu: 0.1, cpuMax: 2.1, memory: 0, memoryUsage: 0 },
        { ts: 1, cpu: 0.2, memory: 0, memoryUsage: 0 },
      ],
    });
    expect(wrapper.text()).toContain("peak 2.1%");
    expect(wrapper.text()).toContain("CPU");
    const bars = wrapper.findComponent(BarChartStub);
    expect(bars.props("max")).toBeUndefined();
    expect(bars.props("chartData").map((b: { percent: number }) => b.percent)).toEqual([0.1, 0.2]);
  });

  test("CPU over time names the window's ends, and the hovered bar's moment and value", async () => {
    const day = 24 * 3600_000;
    const from = Date.UTC(2026, 10, 12, 12);
    const wrapper = render({
      type: "cpu_over_time",
      container: "api",
      from,
      to: from + 7 * day,
      points: [
        { ts: from, cpu: 5, memory: 0, memoryUsage: 0 },
        { ts: from + 5 * day, cpu: 6.5, cpuMax: 58, memory: 0, memoryUsage: 0 },
      ],
    });
    const axis = wrapper.find("[data-range]");
    expect(axis.text()).toContain("Nov 12");
    expect(axis.text()).toContain("Nov 19");
    expect(wrapper.find("[data-moment]").exists()).toBe(false);

    const bars = wrapper.findComponent(BarChartStub);
    bars.vm.$emit("hoverValue", 6.5, 1, 2, 58);
    await wrapper.vm.$nextTick();
    const moment = wrapper.find("[data-moment]").text();
    expect(moment).toContain("Nov 17");
    expect(moment).toContain("6.5%");
    expect(moment).toContain("peak 58.0%");

    bars.vm.$emit("hoverEnd");
    await wrapper.vm.$nextTick();
    expect(wrapper.find("[data-moment]").exists()).toBe(false);
  });

  test("CPU over time with no series draws no chart", () => {
    const wrapper = render({ type: "cpu_over_time", container: "rook", from: 0, to: 1, points: [] });
    expect(wrapper.findComponent(BarChartStub).exists()).toBe(false);
  });

  test("pattern count: the total and cloud's buckets as they came", () => {
    const volume = [{ ts: 0, level: "warn", count: 12 }];
    const wrapper = render({
      type: "pattern_count",
      container: "rook",
      query: "Invalid HTTP",
      from: 0,
      to: 1,
      total: 147,
      volume,
      stepNs: 3.6e12,
    });
    expect(wrapper.text()).toContain("matching lines: 147");
    expect(wrapper.findComponent(VolumeStub).props("buckets")).toEqual(volume);
  });

  test("ranking: one row per container, widths against the top value", () => {
    const wrapper = render({
      type: "container_ranking",
      metric: "restarts",
      from: 0,
      to: 1,
      rows: [
        { container: "rook", value: 24 },
        { container: "sonarr", value: 6 },
      ],
    });
    expect(wrapper.text()).toContain("Top containers by restarts");
    const widths = wrapper.findAll("li .bg-primary").map((b) => (b.element as HTMLElement).style.width);
    expect(widths).toEqual(["100%", "25%"]);
  });

  test("container events with nothing recorded says so", () => {
    const wrapper = render({
      type: "container_events",
      container: "rook",
      from: 0,
      to: 1,
      facts: { exits: 0, ooms: 0, restarts: 0, eventsRecorded: true, capped: false },
    });
    expect(wrapper.text()).toContain("No exits or restarts recorded");
  });

  test("logs: one row per line", () => {
    const wrapper = render({
      type: "logs",
      container: "rook",
      from: 0,
      to: 1,
      lines: [
        { timestamp: "2026-11-15T19:38:14Z", level: "warn", message: "WARNING: Invalid HTTP request received." },
        { level: "error", message: "boom" },
      ],
    });
    expect(wrapper.findAll("li")).toHaveLength(2);
    expect(wrapper.text()).toContain("WARNING: Invalid HTTP request received.");
  });

  // A window that ended in the past ends its axis there, not at now: padded to
  // now, a week-old count drops off the strip while its total still says 147.
  test("pattern count: the volume axis ends at the window's end", () => {
    const wrapper = render({
      type: "pattern_count",
      container: "rook",
      query: "x",
      from: 1_000,
      to: 9_000,
      total: 3,
      volume: [{ ts: 1e9, level: "warn", count: 3 }],
      stepNs: 1e9,
    });
    expect(wrapper.findComponent(VolumeStub).props("end")).toBe(9_000);
  });

  // Without stored events "nothing recorded" is not known; say why instead.
  test("container events without stored events says none are stored, not none happened", () => {
    const wrapper = render({
      type: "container_events",
      container: "rook",
      from: 0,
      to: 1,
      facts: { exits: 0, ooms: 0, restarts: 0, eventsRecorded: false, capped: false },
    });
    expect(wrapper.text()).toContain("No events are stored for this container");
    expect(wrapper.text()).not.toContain("No exits or restarts recorded");
  });

  test("container events at the row cap says the counts are a floor", () => {
    const wrapper = render({
      type: "container_events",
      container: "rook",
      from: 0,
      to: 1,
      facts: { exits: 500, ooms: 0, restarts: 0, eventsRecorded: true, capped: true },
    });
    expect(wrapper.text()).toContain("Older events were cut off");
  });
});
