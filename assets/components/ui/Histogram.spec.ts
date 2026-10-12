/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test } from "vitest";
import Histogram from "./Histogram.vue";
import type { HistogramBin } from "./histogramBins";

const bins: HistogramBin[] = [
  { start: 0, end: 9, count: 3 },
  { start: 10, end: 19, count: 0 },
  { start: 20, end: 29, count: 7 },
];

describe("<Histogram />", () => {
  test("emits the hovered bin, then hoverEnd on leave", async () => {
    const wrapper = mount(Histogram, { props: { bins } });
    const columns = wrapper.findAll('[data-testid="histogram-bin"]');

    await columns[2].trigger("mouseenter");
    expect(wrapper.emitted("hoverBin")).toEqual([[bins[2]]]);

    await wrapper.find("div.h-28").trigger("mouseleave");
    expect(wrapper.emitted("hoverEnd")).toEqual([[]]);
  });
});
