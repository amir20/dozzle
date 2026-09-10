/**
 * @vitest-environment jsdom
 */
import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { nextTick } from "vue";
import BarChart, { type BarDataPoint } from "./BarChart.vue";

// useElementSize relies on ResizeObserver which jsdom lacks, so the width stays
// 0 and the chart never renders. Mock it with a controllable width ref that we
// flip to a real value after mount to mimic the ResizeObserver firing.
const holder = vi.hoisted(() => ({ width: null as ReturnType<typeof import("vue").ref<number>> | null }));
vi.mock("@vueuse/core", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@vueuse/core")>();
  const { ref: vueRef } = await import("vue");
  holder.width = vueRef(0);
  return { ...actual, useElementSize: () => ({ width: holder.width, height: vueRef(0) }) };
});

function ramp(start = 0, n = 300): BarDataPoint[] {
  return Array.from({ length: n }, (_, i) => ({ percent: start + i, value: start + i }));
}

function constant(percent: number, n = 300): BarDataPoint[] {
  return Array.from({ length: n }, () => ({ percent, value: percent }));
}

function heightOf(wrapper: ReturnType<typeof mount>, index: number): number {
  const style = wrapper.findAll(".bar")[index]?.attributes("style") ?? "";
  const match = style.match(/--height:\s*([\d.]+)%/);
  return match ? parseFloat(match[1]) : 0;
}

async function mountAndRender(chartData: BarDataPoint[]) {
  // Mount with an unmeasured element, then simulate ResizeObserver reporting a
  // real width -> triggers the initial recalculate, like the live component.
  holder.width!.value = 0;
  const wrapper = mount(BarChart, { props: { chartData } });
  await nextTick();
  holder.width!.value = 300;
  await nextTick();
  await flushPromises();
  return wrapper;
}

describe("<BarChart />", () => {
  test("exposed recalculate() rebuilds all bars after a wholesale data swap", async () => {
    // First container: a ramp where the oldest bars are near zero.
    const wrapper = await mountAndRender(ramp());
    expect(heightOf(wrapper, 0)).toBeLessThan(20); // oldest ramp bar is tiny

    // A stat tick arrives: the rolling window shifts by one, marking the chart
    // initialized so further changes only patch the last bar.
    await wrapper.setProps({ chartData: ramp(1) });
    await nextTick();

    // Switch containers: the whole series is replaced with a flat high value.
    // The chart caches bars and only patches the last one, so without help the
    // older bars stay stale.
    await wrapper.setProps({ chartData: constant(1000) });
    await nextTick();
    expect(heightOf(wrapper, 0)).toBeLessThan(20); // still stale

    // The parent owns container switches and calls recalculate() to refresh.
    (wrapper.vm as unknown as { recalculate: () => void }).recalculate();
    await nextTick();
    expect(heightOf(wrapper, 0)).toBeGreaterThan(50); // flat series -> uniform height
  });

  test("renders downsampled bars once width is known", async () => {
    const wrapper = await mountAndRender(constant(1000));
    expect(wrapper.findAll(".bar").length).toBeGreaterThan(0);
    expect(heightOf(wrapper, 0)).toBeGreaterThan(50);
  });
});
