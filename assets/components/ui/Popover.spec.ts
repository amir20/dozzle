/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ref } from "vue";
import Popover from "./Popover.vue";

const popover = vi.hoisted(() => ({ show: vi.fn(), hide: vi.fn(), toggle: vi.fn() }));
vi.mock("@/composable/ui/popover", () => ({
  useAnchoredPopover: () => ({ isOpen: ref(false), onBeforeToggle: vi.fn(), onToggle: vi.fn(), ...popover }),
}));
// A touchscreen laptop reports `(hover: hover)` too.
vi.mock("@/composable/ui/media", () => ({ canHover: ref(true), isMobile: ref(false) }));

const anchor = () =>
  mount(Popover, { props: { hover: true }, slots: { trigger: "<button />" } }).find(".popover-anchor");

describe("<Popover hover />", () => {
  beforeEach(() => vi.useFakeTimers());

  test("a mouse opens it on hover and closes it on leave", async () => {
    const a = anchor();
    await a.trigger("pointerenter", { pointerType: "mouse" });
    expect(popover.show).toHaveBeenCalled();
    await a.trigger("pointerleave", { pointerType: "mouse" });
    vi.runAllTimers();
    expect(popover.hide).toHaveBeenCalled();
  });

  // #5748: a tap fires enter on touch-down and leave on touch-up, so the menu shut itself.
  test("a tap toggles it and lifting the finger does not close it", async () => {
    vi.clearAllMocks();
    const a = anchor();
    await a.trigger("pointerenter", { pointerType: "touch" });
    await a.trigger("pointerleave", { pointerType: "touch" });
    await a.trigger("click");
    vi.runAllTimers();
    expect(popover.show).not.toHaveBeenCalled();
    expect(popover.hide).not.toHaveBeenCalled();
    expect(popover.toggle).toHaveBeenCalledOnce();
  });
});
