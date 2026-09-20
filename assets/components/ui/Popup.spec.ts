/**
 * @vitest-environment jsdom
 */
import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { h } from "vue";
import Popup from "./Popup.vue";

// jsdom has no popover API, so the panel cannot actually open. Stubbing the two
// calls is enough: what matters here is what is in the DOM when showPopover runs.
const contentAtShow: boolean[] = [];

beforeEach(() => {
  contentAtShow.length = 0;
  vi.useFakeTimers();
  (HTMLElement.prototype as unknown as { showPopover: () => void }).showPopover = function (this: HTMLElement) {
    contentAtShow.push(!!this.querySelector(".panel-content"));
  };
  (HTMLElement.prototype as unknown as { hidePopover: () => void }).hidePopover = () => {};
});

afterEach(() => vi.useRealTimers());

function mountPopup() {
  return mount(Popup, {
    slots: {
      default: () => h("button", { class: "row" }, "row"),
      content: () => h("div", { class: "panel-content" }, "details"),
    },
  });
}

async function hoverRow(wrapper: ReturnType<typeof mountPopup>) {
  // The anchor listener is attached by a post-flush watcher, so a pointerenter
  // dispatched in the same tick as the mount lands before anything is listening.
  await flushPromises();
  await wrapper.find(".row").trigger("pointerenter");
  // Past OPEN_DELAY, then let the awaited nextTick inside the open path settle.
  await vi.advanceTimersByTimeAsync(1100);
  await flushPromises();
}

describe("<Popup />", () => {
  // The sidebar renders one of these per container, so mounting the panel with the
  // row meant a live component tree per container for panels nobody had opened.
  test("does not render its content until the row is hovered", () => {
    const wrapper = mountPopup();
    expect(wrapper.find(".row").exists()).toBe(true);
    expect(wrapper.find(".panel-content").exists()).toBe(false);
  });

  test("renders the content on hover", async () => {
    const wrapper = mountPopup();
    await hoverRow(wrapper);
    expect(wrapper.find(".panel-content").exists()).toBe(true);
  });

  // The panel is measured and positioned immediately after showPopover(), so an
  // empty box would be placed against the wrong size on the first frame.
  test("mounts the content before showing the panel", async () => {
    const wrapper = mountPopup();
    await hoverRow(wrapper);
    expect(contentAtShow).toEqual([true]);
  });

  // Deliberate: a row hovered once tends to be hovered again, and remounting per
  // hover would redo the work this defers.
  test("keeps the content mounted after the pointer leaves", async () => {
    const wrapper = mountPopup();
    await hoverRow(wrapper);

    await wrapper.find(".row").trigger("pointerleave");
    await vi.advanceTimersByTimeAsync(500);
    await flushPromises();

    expect(wrapper.find(".panel-content").exists()).toBe(true);
  });
});
