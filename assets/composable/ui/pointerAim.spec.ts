/**
 * @vitest-environment jsdom
 */
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { AIM_IDLE, aimAt, aimOwner, clearAim } from "./pointerAim";

// A panel to the right of a 200px wide sidebar, 10px clear of it.
function panelAt(left = 210, top = 0, height = 300) {
  const el = document.createElement("div");
  el.getBoundingClientRect = () => ({ left, right: left + 256, top, bottom: top + height }) as DOMRect;
  return el;
}

const move = (clientX: number, clientY: number) =>
  window.dispatchEvent(new MouseEvent("pointermove", { clientX, clientY }));

let panel: HTMLElement;

beforeEach(() => {
  vi.useFakeTimers();
  panel = panelAt();
  // Left the row at its right edge, a third of the way down the panel.
  aimAt(panel, new MouseEvent("pointerleave", { clientX: 200, clientY: 100 }), panel);
});

afterEach(() => {
  clearAim(panel);
  vi.useRealTimers();
});

describe("pointerAim", () => {
  test("moving diagonally toward the panel keeps aiming at it", () => {
    move(204, 130);
    expect(aimOwner()).toBe(panel);
  });

  test("moving away from the panel drops the aim", () => {
    move(150, 130);
    expect(aimOwner()).toBeNull();
  });

  test("resting inside the triangle drops the aim", () => {
    move(204, 130);
    vi.advanceTimersByTime(AIM_IDLE + 1);
    expect(aimOwner()).toBeNull();
  });

  test("clearing it as the owner drops the aim", () => {
    move(204, 130);
    clearAim(panel);
    expect(aimOwner()).toBeNull();
  });
});
