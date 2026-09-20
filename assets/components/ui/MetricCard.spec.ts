/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test } from "vitest";
import { createI18n } from "vue-i18n";
import { defineComponent, h, nextTick } from "vue";
import MetricCard from "./MetricCard.vue";
import type { BarDataPoint } from "./BarChart.vue";

// The real chart needs a measured element to draw anything, and what is under
// test here is the readout above it, so the chart is reduced to the one thing
// this card listens to.
const BarChartStub = defineComponent({
  name: "BarChart",
  props: ["chartData", "barClass", "max"],
  emits: ["hoverValue", "hoverEnd"],
  setup: () => () => h("div", { class: "bar-chart-stub" }),
});

const i18n = createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false, messages: { en: {} } });

const Icon = defineComponent({ setup: () => () => h("span") });

// 60 points one second apart: the newest is index 59.
const chartData: BarDataPoint[] = Array.from({ length: 60 }, (_, i) => ({ percent: i, value: i }));

function mountCard(props: Record<string, unknown> = {}) {
  return mount(MetricCard, {
    props: {
      label: "CPU",
      capacity: "2 cores",
      icon: Icon,
      value: 58.4,
      chartData,
      formatValue: (v: number) => `${v.toFixed(1)}%`,
      ...props,
    },
    global: { plugins: [i18n], stubs: { BarChart: BarChartStub } },
  });
}

async function hover(wrapper: ReturnType<typeof mountCard>, index: number, bars: number) {
  const point = chartData[Math.min(chartData.length - 1, Math.floor(((index + 0.5) * chartData.length) / bars))];
  wrapper.findComponent(BarChartStub).vm.$emit("hoverValue", point.value, index, bars);
  await nextTick();
}

describe("MetricCard", () => {
  test("shows the live value with avg and peak under it", () => {
    const wrapper = mountCard();
    expect(wrapper.text()).toContain("58.4%");
    expect(wrapper.text()).toContain("avg 29.5% · pk 59.0%");
  });

  test("a hovered bar replaces the value and says when it was", async () => {
    const wrapper = mountCard({ sampleInterval: 1000 });
    // 30 bars over 60 points: bar 0 covers the two oldest, so it is ~58s back.
    await hover(wrapper, 0, 30);

    expect(wrapper.text()).toContain("1.0%");
    expect(wrapper.text()).not.toContain("avg");
    expect(wrapper.text()).toContain("1 min. ago");
  });

  test("the newest bar reads as now", async () => {
    const wrapper = mountCard({ sampleInterval: 1000 });
    await hover(wrapper, 29, 30);

    expect(wrapper.text()).toContain("59.0%");
    expect(wrapper.text()).toContain("now");
  });

  test("leaving the chart restores the live readout", async () => {
    const wrapper = mountCard({ sampleInterval: 1000 });
    await hover(wrapper, 10, 30);
    expect(wrapper.text()).not.toContain("avg");

    wrapper.findComponent(BarChartStub).vm.$emit("hoverEnd");
    await nextTick();
    expect(wrapper.text()).toContain("58.4%");
    expect(wrapper.text()).toContain("avg 29.5% · pk 59.0%");
  });

  // Without a cadence from the parent a bar has no time to name, so the card
  // keeps the value swap and leaves avg/pk in place rather than inventing one.
  test("no sample interval keeps avg and peak while hovering", async () => {
    const wrapper = mountCard();
    await hover(wrapper, 0, 30);

    expect(wrapper.text()).toContain("1.0%");
    expect(wrapper.text()).toContain("avg 29.5% · pk 59.0%");
  });
});
