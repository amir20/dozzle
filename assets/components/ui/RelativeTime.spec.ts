/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { afterAll, beforeEach, describe, expect, test, vi } from "vitest";
import { defineComponent, h, nextTick } from "vue";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { hosts: [], base: "", profile: {}, user: null, authProvider: "none" },
  withBase: (path: string) => path,
}));

// Order matters. The ticker starts its interval at module scope, so fake timers have to be
// installed before the component is imported for the tick to be advanceable -- and the
// counter has to wrap the fake setInterval that useFakeTimers just installed, since
// wrapping the real one first would just get clobbered.
vi.useFakeTimers();
vi.setSystemTime(new Date("2026-09-03T12:00:00Z"));

let intervals = 0;
const fakeSetInterval = globalThis.setInterval;
globalThis.setInterval = ((...args: Parameters<typeof setInterval>) => {
  intervals++;
  return fakeSetInterval(...args);
}) as typeof setInterval;

const RelativeTime = (await import("./RelativeTime.vue")).default;

// The module-level interval is the only one the whole app should ever create.
const intervalsAfterImport = intervals;

afterAll(() => {
  vi.useRealTimers();
});

beforeEach(() => {
  vi.setSystemTime(new Date("2026-09-03T12:00:00Z"));
});

function mountMany(n: number, date: Date) {
  return mount(
    defineComponent({
      render: () =>
        h(
          "div",
          Array.from({ length: n }, (_, i) => h(RelativeTime, { key: i, date })),
        ),
    }),
  );
}

describe("RelativeTime", () => {
  test("renders a relative timestamp", () => {
    const wrapper = mount(RelativeTime, { props: { date: new Date("2026-09-03T11:00:00Z") } });
    expect(wrapper.text()).not.toBe("");
    expect(wrapper.find("time").attributes("datetime")).toBe("2026-09-03T11:00:00.000Z");
  });

  test("re-renders when the date prop changes", async () => {
    const wrapper = mount(RelativeTime, { props: { date: new Date("2026-09-03T11:59:00Z") } });
    const before = wrapper.text();

    await wrapper.setProps({ date: new Date("2026-09-01T12:00:00Z") });
    expect(wrapper.text()).not.toBe(before);
  });

  test("the shared tick refreshes text as time passes", async () => {
    const wrapper = mount(RelativeTime, { props: { date: new Date("2026-09-03T12:00:00Z") } });
    const before = wrapper.text();

    vi.advanceTimersByTime(60 * 60 * 1000);
    await nextTick();

    expect(wrapper.text()).not.toBe(before);
  });

  test("all instances share one timer", async () => {
    // Regression guard: useIntervalFn used to live in setup, so the container table stood
    // up one setInterval per row and every instance woke on its own unaligned schedule.
    intervals = 0;
    const wrapper = mountMany(50, new Date("2026-09-03T11:00:00Z"));
    await nextTick();

    expect(intervals).toBe(0);
    // ...because the one shared timer was already started when the ticker was imported.
    expect(intervalsAfterImport).toBe(1);

    wrapper.unmount();
  });
});
