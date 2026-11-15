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
const VolumeStub = defineComponent({ props: ["buckets", "stepNs"], template: "<div data-volume />" });

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
              "cloud-chat": {
                visual: {
                  memory: "Memory",
                  "peak-of-limit": "peak {peak} of {limit}",
                  "no-events": "No exits or restarts recorded",
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
});
