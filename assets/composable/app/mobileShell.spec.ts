/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import { defineComponent, ref } from "vue";

import { isLogRoute, useEdgeSwipeBack } from "./mobileShell";

const Blank = defineComponent({ template: "<div/>" });

function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: "/", name: "/", component: Blank },
      { path: "/notifications", name: "/notifications", component: Blank },
      { path: "/container/:id", name: "/container/[id]", component: Blank },
    ],
  });
}

function setStandalone(value: boolean | undefined) {
  Object.defineProperty(navigator, "standalone", { value, configurable: true });
}

async function mountSwipe(enabled = true) {
  const router = makeRouter();
  await router.replace("/container/abc");
  await router.isReady();
  const push = vi.spyOn(router, "push");
  const back = vi.spyOn(router, "back").mockImplementation(() => {});

  const on = ref(enabled);
  const wrapper = mount(
    defineComponent({
      setup: () => useEdgeSwipeBack(on),
      template: "<div/>",
    }),
    { global: { plugins: [router] } },
  );
  return { push, back, on, wrapper };
}

// jsdom has no Touch constructor, and the handlers only read coordinates.
function touch(type: "touchstart" | "touchend", x: number, y: number) {
  const point = { clientX: x, clientY: y };
  window.dispatchEvent(Object.assign(new Event(type), { touches: [point], changedTouches: [point] }));
}

function swipe(from: [number, number], to: [number, number]) {
  touch("touchstart", ...from);
  touch("touchend", ...to);
}

describe("isLogRoute", () => {
  test("treats a log view as a pushed screen and the tab pages as not", async () => {
    const router = makeRouter();
    await router.push("/container/abc");
    expect(isLogRoute(router.currentRoute.value)).toBe(true);
    await router.push("/notifications");
    expect(isLogRoute(router.currentRoute.value)).toBe(false);
    await router.push("/");
    expect(isLogRoute(router.currentRoute.value)).toBe(false);
  });
});

describe("edge swipe back", () => {
  let wrapper: ReturnType<typeof mount> | undefined;

  beforeEach(() => {
    setStandalone(true);
    window.history.replaceState(null, "");
  });

  afterEach(() => {
    wrapper?.unmount();
    wrapper = undefined;
    setStandalone(undefined);
  });

  test("goes home from the left edge when there is no in-app history", async () => {
    const m = await mountSwipe();
    wrapper = m.wrapper;
    swipe([5, 300], [150, 310]);
    expect(m.push).toHaveBeenCalledWith({ name: "/" });
    expect(m.back).not.toHaveBeenCalled();
  });

  test("goes back through history when there is somewhere to go", async () => {
    window.history.replaceState({ back: "/" }, "");
    const m = await mountSwipe();
    wrapper = m.wrapper;
    swipe([5, 300], [150, 310]);
    expect(m.back).toHaveBeenCalledOnce();
  });

  test.each([
    ["starts away from the edge", [40, 300], [200, 300]],
    ["is too short", [5, 300], [60, 300]],
    ["is mostly vertical", [5, 300], [110, 420]],
  ] as const)("ignores a swipe that %s", async (_, from, to) => {
    const m = await mountSwipe();
    wrapper = m.wrapper;
    swipe([...from], [...to]);
    expect(m.push).not.toHaveBeenCalled();
    expect(m.back).not.toHaveBeenCalled();
  });

  test("does nothing while disabled", async () => {
    const m = await mountSwipe(false);
    wrapper = m.wrapper;
    swipe([5, 300], [150, 300]);
    expect(m.push).not.toHaveBeenCalled();
  });

  test("stays off outside an installed iOS app, where the platform already has the gesture", async () => {
    setStandalone(undefined);
    const m = await mountSwipe();
    wrapper = m.wrapper;
    swipe([5, 300], [150, 300]);
    expect(m.push).not.toHaveBeenCalled();
    expect(m.back).not.toHaveBeenCalled();
  });
});
