/**
 * @vitest-environment jsdom
 */
import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { nextTick } from "vue";
import BarChart, { type BarDataPoint } from "./BarChart.vue";

// Mirror the component's own layout constants, which set the column pitch.
const GAP = 2;
const CHART_WIDTH = 300;

// useElementSize relies on ResizeObserver which jsdom lacks, so the width stays
// 0 and the chart never renders. Mock it with a controllable width ref that we
// flip to a real value after mount to mimic the ResizeObserver firing.
const holder = vi.hoisted(() => ({
  width: null as ReturnType<typeof import("vue").ref<number>> | null,
  height: null as ReturnType<typeof import("vue").ref<number>> | null,
}));
vi.mock("@vueuse/core", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@vueuse/core")>();
  const { ref: vueRef } = await import("vue");
  holder.width = vueRef(0);
  // Bars are drawn in real pixels, so the chart needs a height to draw into. 100
  // keeps every assertion below reading on the same 0-100 scale as before. Written
  // out rather than referencing CHART_HEIGHT: vi.mock hoists this factory above the
  // consts, so naming one here is a use-before-init.
  holder.height = vueRef(100);
  return { ...actual, useElementSize: () => ({ width: holder.width, height: holder.height }) };
});

function ramp(start = 0, n = 300): BarDataPoint[] {
  return Array.from({ length: n }, (_, i) => ({ percent: start + i, value: start + i }));
}

function constant(percent: number, n = 300): BarDataPoint[] {
  return Array.from({ length: n }, () => ({ percent, value: percent }));
}

// Each layer is a single <path> whose bars are one `M x bottomV top` subpath each,
// so this reads the drawn geometry straight back out rather than trusting internals.
// Bars come back in left-to-right order across both layers, as they are drawn.
type DrawnBar = { x: number; height: number; sampled: boolean };
function bars(wrapper: ReturnType<typeof mount>): DrawnBar[] {
  const layer = (kind: string, sampled: boolean): DrawnBar[] => {
    const d = wrapper.find(`[data-bars="${kind}"]`).exists()
      ? (wrapper.find(`[data-bars="${kind}"]`).attributes("d") ?? "")
      : "";
    return [...d.matchAll(/M([\d.]+) ([\d.]+)V([\d.]+)/g)].map((m) => ({
      x: parseFloat(m[1]),
      height: parseFloat(m[2]) - parseFloat(m[3]),
      sampled,
    }));
  };
  return [...layer("padding", false), ...layer("sampled", true)].sort((a, b) => a.x - b.x);
}

function heightOf(wrapper: ReturnType<typeof mount>, index: number): number {
  return bars(wrapper)[index]?.height ?? 0;
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

  // The svg must never take part in sizing its parent. An svg in flow is a replaced
  // element: `width`/`height` attributes give it an intrinsic width and a `viewBox`
  // alone still gives it an intrinsic ratio, either of which becomes the min-content
  // width of the `flex-1` cell it lives in. That width is measured from the cell, so it
  // became a floor the cell could not shrink below, and the column ratcheted wider
  // every time the readout beside it changed width. jsdom does no layout, so this
  // guards the two properties that caused it rather than the width itself.
  test("the svg contributes nothing to its parent's width", async () => {
    const wrapper = await mountAndRender(constant(1000));
    const svg = wrapper.find("svg");

    expect(svg.attributes("width")).toBeUndefined();
    expect(svg.attributes("height")).toBeUndefined();
    expect(svg.classes()).toContain("absolute");
    expect(wrapper.classes()).toContain("min-w-0");
  });

  test("renders downsampled bars once width is known", async () => {
    const wrapper = await mountAndRender(constant(1000));
    expect(bars(wrapper).length).toBeGreaterThan(0);
    expect(heightOf(wrapper, 0)).toBeGreaterThan(50);
  });

  // A handful of points across a wide element makes each column, and so the stroke
  // drawn for it, far wider than the element is tall. The cap allowance subtracted
  // from the drawable height is half a stroke, so unbounded it went past the whole
  // height and every bar came out at zero. The cloud rail asks for whatever the API
  // returns for the selected window, so this is reachable with real data.
  test("a sparse series in a short chart still draws visible bars", async () => {
    holder.height!.value = 16; // h-4, as the container table renders it
    const wrapper = await mountAndRender(constant(50, 4));

    const drawn = bars(wrapper);
    expect(drawn).toHaveLength(4);
    for (const bar of drawn) expect(bar.height).toBeGreaterThan(0);

    holder.height!.value = 100;
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

    const heights = () => bars(wrapper).map((b) => b.height);
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
  // jsdom lays nothing out, so the chart reports a zero rect. Only `left` is read
  // from it now: the column pitch comes from the mocked useElementSize width, the
  // same source the guide uses.
  function layOutBars(wrapper: ReturnType<typeof mount>) {
    vi.spyOn(wrapper.element, "getBoundingClientRect").mockReturnValue({
      left: 0,
      right: CHART_WIDTH,
      width: CHART_WIDTH,
    } as DOMRect);
  }

  // The x at the centre of bar `index`, using the component's own pitch formula.
  function xOfBar(wrapper: ReturnType<typeof mount>, index: number) {
    const count = bars(wrapper).length;
    return (index + 0.5) * ((CHART_WIDTH + GAP) / count);
  }

  test("a mouse move reports the bar under the pointer", async () => {
    const wrapper = await mountAndRender(ramp());
    layOutBars(wrapper);
    await wrapper.trigger("mousemove", { clientX: xOfBar(wrapper, 2) });

    const emitted = wrapper.emitted("hoverValue");
    expect(emitted).toHaveLength(1);
    expect(emitted![0][1]).toBe(2);
  });

  // A touch screen never fires mousemove, so the history panel's readout was
  // stuck on the peak and no bar could be read on a phone.
  test("a touch reports the bar under the finger", async () => {
    const wrapper = await mountAndRender(ramp());
    layOutBars(wrapper);
    await wrapper.trigger("touchstart", { touches: [{ clientX: xOfBar(wrapper, 2) }] });
    await wrapper.trigger("touchmove", { touches: [{ clientX: xOfBar(wrapper, 5) }] });

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

    const drawn = bars(wrapper);
    expect(drawn[0].sampled).toBe(false);
    expect(drawn.at(-1)!.sampled).toBe(true);

    layOutBars(wrapper);
    await wrapper.trigger("mousemove", { clientX: xOfBar(wrapper, 0) });
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

    const firstSampled = bars(wrapper).findIndex((b) => b.sampled);
    expect(firstSampled).toBe(59);

    layOutBars(wrapper);
    await wrapper.trigger("mousemove", { clientX: xOfBar(wrapper, firstSampled) });
    const emitted = wrapper.emitted("hoverValue");
    expect(emitted).toHaveLength(1);
    // 90, the measurement. Averaging the padded zero in would report 72.
    expect(emitted![0][0]).toBe(90);
  });

  // Queried by `data-guide` rather than `.absolute`: the svg is positioned as well, so
  // the utility class no longer picks out the guide on its own.
  test("a hovered sampled bar draws a guide on its column", async () => {
    const wrapper = await mountAndRender(ramp());
    layOutBars(wrapper);
    expect(wrapper.find("[data-guide]").exists()).toBe(false);

    await wrapper.trigger("mousemove", { clientX: xOfBar(wrapper, 2) });
    expect(wrapper.find("[data-guide]").exists()).toBe(true);

    await wrapper.trigger("mouseleave");
    expect(wrapper.find("[data-guide]").exists()).toBe(false);
  });

  // The pointer sits over a fixed column while the series scrolls underneath, so
  // the readout has to follow the column. It used to freeze on the sample that was
  // there when the pointer stopped moving.
  test("the readout follows the hovered column as the series scrolls", async () => {
    const point = (v: number) => ({ percent: v, value: v });
    let series: BarDataPoint[] = Array.from({ length: 300 }, () => point(10));
    holder.width!.value = 0;
    const wrapper = mount(BarChart, { props: { chartData: series } });
    await nextTick();
    holder.width!.value = CHART_WIDTH;
    await nextTick();
    await flushPromises();
    layOutBars(wrapper);

    const count = bars(wrapper).length;
    const last = count - 1;
    await wrapper.trigger("mousemove", { clientX: xOfBar(wrapper, last) });
    expect(wrapper.emitted("hoverValue")!.at(-1)![0]).toBe(10);

    // A tick of the live series pushes a very different sample into that column.
    series = [...series.slice(1), point(90)];
    await wrapper.setProps({ chartData: series });
    await nextTick();

    expect(wrapper.emitted("hoverValue")!.at(-1)![0]).not.toBe(10);
  });

  // Swapping the chart out is a silent way to leave it: no mouseleave fires, and a
  // consumer would sit on the last hovered number for good.
  test("unmounting releases the hover", async () => {
    const wrapper = await mountAndRender(ramp());
    layOutBars(wrapper);
    await wrapper.trigger("mousemove", { clientX: xOfBar(wrapper, 2) });
    expect(wrapper.emitted("hoverEnd")).toBeUndefined();

    wrapper.unmount();
    expect(wrapper.emitted("hoverEnd")).toHaveLength(1);
  });

  // The parent owns wholesale series swaps: HostCard's backfill keeps the length at
  // 300 and can leave the last entry untouched, so the data watcher never fires and
  // only the forced recalculate can refresh what the pointer is reading.
  test("a forced recalculate re-reports the hovered bar", async () => {
    const point = (v: number) => ({ percent: v, value: v });
    // Shared last entry, so `chartData.at(-1)` keeps its identity across the swap.
    const tail = point(10);
    let series: BarDataPoint[] = [...Array.from({ length: 299 }, () => point(10)), tail];

    holder.width!.value = 0;
    const wrapper = mount(BarChart, { props: { chartData: series } });
    await nextTick();
    holder.width!.value = CHART_WIDTH;
    await nextTick();
    await flushPromises();
    layOutBars(wrapper);

    await wrapper.trigger("mousemove", { clientX: xOfBar(wrapper, 10) });
    expect(wrapper.emitted("hoverValue")!.at(-1)![0]).toBe(10);

    series = [...Array.from({ length: 299 }, () => point(90)), tail];
    await wrapper.setProps({ chartData: series });
    await nextTick();
    // Nothing in the chart can see this swap, which is why the contract exists.
    expect(wrapper.emitted("hoverValue")!.at(-1)![0]).toBe(10);

    (wrapper.vm as unknown as { recalculate: () => void }).recalculate();
    await nextTick();
    expect(wrapper.emitted("hoverValue")!.at(-1)![0]).toBe(90);
  });

  // A resize re-buckets the series, so the column the pointer was on may not exist
  // any more. There is no clientX to re-run the hit test with until it moves again.
  test("a resize releases a hover that is no longer on a bar", async () => {
    const wrapper = await mountAndRender(ramp());
    layOutBars(wrapper);
    const count = bars(wrapper).length;
    await wrapper.trigger("mousemove", { clientX: xOfBar(wrapper, count - 1) });
    expect(wrapper.find("[data-guide]").exists()).toBe(true);

    holder.width!.value = CHART_WIDTH / 4;
    await nextTick();
    await flushPromises();

    expect(wrapper.emitted("hoverEnd")).toHaveLength(1);
    expect(wrapper.find("[data-guide]").exists()).toBe(false);
  });

  test("a touch with no contact point reports nothing", async () => {
    const wrapper = await mountAndRender(ramp());
    layOutBars(wrapper);
    await wrapper.trigger("touchend", { touches: [] });
    await wrapper.trigger("touchstart", { touches: [] });

    expect(wrapper.emitted("hoverValue")).toBeUndefined();
  });
});
