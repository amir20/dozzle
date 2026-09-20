/**
 * @vitest-environment jsdom
 */
import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { nextTick } from "vue";
import BarChart, { type BarDataPoint } from "./BarChart.vue";

// Mirrors the gap between bars in the component, which sets the column pitch.
const GAP = 2;

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

describe("BarChart stability", () => {
  // The whole point of the changeCounter: a recalculation is only phase-stable
  // every `bucketSize` ticks, when the window has shifted by exactly one bucket.
  // Recalculating off-phase re-buckets against data that moved a fraction of a
  // bar, so every bar jumps instead of the chart scrolling left by one.
  test("advancing the padding boundary does not re-bucket mid-scroll", async () => {
    const point = (v: number) => ({ percent: v, value: v });
    // 300 points, the first 252 of them padding. At 300px the chart draws 60 bars
    // of 5 points each, so the boundary crosses a bucket when it passes 250.
    let series: BarDataPoint[] = [
      ...Array.from({ length: 252 }, () => point(0)),
      ...Array.from({ length: 48 }, () => point(90)),
    ];
    let boundary = 252;

    holder.width!.value = 0;
    const wrapper = mount(BarChart, { props: { chartData: series, sampledFrom: boundary } });
    await nextTick();
    holder.width!.value = 300;
    await nextTick();
    await flushPromises();

    const heights = () => wrapper.findAll(".bar").map((_, i) => heightOf(wrapper, i));
    const tick = async () => {
      series = [...series.slice(1), point(90)];
      boundary = Math.max(0, boundary - 1);
      await wrapper.setProps({ chartData: series, sampledFrom: boundary });
      await nextTick();
    };

    // The first data change is BarChart's one-time init, which recalculates by
    // design. Two more leave the boundary at 250, still inside its own bucket.
    await tick();
    await tick();
    const before = heights().slice(0, -1);

    // This tick takes the boundary to 249, crossing into the next bucket, but the
    // window has only moved three points. Nothing behind the last bar may move.
    await tick();
    expect(heights().slice(0, -1)).toEqual(before);
  });
});

describe("BarChart pointer readout", () => {
  // jsdom lays nothing out, so the chart reports a zero rect. Stub the container
  // to a real width: the bars are uniform, so one rect is all the hit test reads.
  // Column pitch is (width + GAP) / count, so this width makes each column `width`.
  function layOutBars(wrapper: ReturnType<typeof mount>, width = 10) {
    const count = wrapper.findAll(".bar").length;
    vi.spyOn(wrapper.element, "getBoundingClientRect").mockReturnValue({
      left: 0,
      right: count * width - GAP,
      width: count * width - GAP,
    } as DOMRect);
  }

  test("a mouse move reports the bar under the pointer", async () => {
    const wrapper = await mountAndRender(ramp());
    layOutBars(wrapper);
    await wrapper.trigger("mousemove", { clientX: 25 });

    const emitted = wrapper.emitted("hoverValue");
    expect(emitted).toHaveLength(1);
    expect(emitted![0][1]).toBe(2);
  });

  // A touch screen never fires mousemove, so the history panel's readout was
  // stuck on the peak and no bar could be read on a phone.
  test("a touch reports the bar under the finger", async () => {
    const wrapper = await mountAndRender(ramp());
    layOutBars(wrapper);
    await wrapper.trigger("touchstart", { touches: [{ clientX: 25 }] });
    await wrapper.trigger("touchmove", { touches: [{ clientX: 55 }] });

    const emitted = wrapper.emitted("hoverValue");
    expect(emitted).toHaveLength(2);
    expect(emitted![0][1]).toBe(2);
    expect(emitted![1][1]).toBe(5);
  });

  // The front of a stats series is padding that keeps the chart full width while
  // real samples scroll in. Reporting it would put a value and a timestamp on a
  // sample nobody took.
  test("padding is drawn faint and never reported", async () => {
    holder.width!.value = 0;
    const wrapper = mount(BarChart, { props: { chartData: ramp(), sampledFrom: 200 } });
    await nextTick();
    holder.width!.value = 300;
    await nextTick();
    await flushPromises();

    const bars = wrapper.findAll(".bar");
    expect(bars[0].classes()).toContain("opacity-15");
    expect(bars.at(-1)!.classes()).toContain("opacity-70");

    layOutBars(wrapper);
    await wrapper.trigger("mousemove", { clientX: 5 });
    expect(wrapper.emitted("hoverValue")).toBeUndefined();
    expect(wrapper.emitted("hoverEnd")).toHaveLength(1);
  });

  // The bucket straddling the boundary holds both padding and real points. Folding
  // the zeros in reported the oldest readable bar well below what was measured.
  test("the boundary bucket averages only its real points", async () => {
    // 296 points of zero padding, then 4 real samples of 90. A 300px chart draws
    // 60 bars, so buckets are 5 points wide and the last one spans 295..299: one
    // padded zero and four measurements of 90.
    const padded: BarDataPoint[] = [
      ...Array.from({ length: 296 }, () => ({ percent: 0, value: 0 })),
      ...Array.from({ length: 4 }, () => ({ percent: 90, value: 90 })),
    ];
    holder.width!.value = 0;
    const wrapper = mount(BarChart, { props: { chartData: padded, sampledFrom: 296 } });
    await nextTick();
    holder.width!.value = 300;
    await nextTick();
    await flushPromises();

    const bars = wrapper.findAll(".bar");
    const firstSampled = bars.findIndex((b) => b.classes().includes("opacity-70"));
    expect(firstSampled).toBe(59);

    layOutBars(wrapper);
    await wrapper.trigger("mousemove", { clientX: firstSampled * 10 + 5 });
    const emitted = wrapper.emitted("hoverValue");
    expect(emitted).toHaveLength(1);
    // 90, the measurement. Averaging the padded zero in would report 72.
    expect(emitted![0][0]).toBe(90);
  });

  test("a hovered sampled bar draws a guide on its column", async () => {
    const wrapper = await mountAndRender(ramp());
    layOutBars(wrapper);
    expect(wrapper.find(".absolute").exists()).toBe(false);

    await wrapper.trigger("mousemove", { clientX: 25 });
    expect(wrapper.find(".absolute").exists()).toBe(true);

    await wrapper.trigger("mouseleave");
    expect(wrapper.find(".absolute").exists()).toBe(false);
  });

  test("a touch with no contact point reports nothing", async () => {
    const wrapper = await mountAndRender(ramp());
    layOutBars(wrapper);
    await wrapper.trigger("touchend", { touches: [] });
    await wrapper.trigger("touchstart", { touches: [] });

    expect(wrapper.emitted("hoverValue")).toBeUndefined();
  });
});
