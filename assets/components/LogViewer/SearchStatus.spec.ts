import { mount } from "@vue/test-utils";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { nextTick } from "vue";
import { createI18n } from "vue-i18n";
import SearchStatus from "./SearchStatus.vue";
import IndeterminateBar from "@/components/common/IndeterminateBar.vue";

/**
 * @vitest-environment jsdom
 */
const i18n = createI18n({
  legacy: false,
  locale: "en",
  messages: {
    en: {
      label: {
        "search-status": {
          searching: "Searching older logs…",
          "searching-to": "Searching older logs… back to {time}",
          capped: "{count} matches · searched back to {time}",
          exhausted: "Searched all logs · {count} matches",
          empty: "No matches · searched all logs",
        },
      },
    },
  },
});

function createStatus(overrides: Record<string, unknown> = {}) {
  return mount(SearchStatus, {
    global: { plugins: [i18n] },
    props: {
      status: { active: false, done: false, matches: 0, scannedTo: undefined, reason: undefined, ...overrides },
    },
  });
}

describe("<SearchStatus />", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  test("stays hidden while a search is active but still fast (flash avoidance)", async () => {
    const wrapper = createStatus({ active: true });
    vi.advanceTimersByTime(100);
    await nextTick();
    expect(wrapper.find("[data-state]").exists()).toBe(false);
  });

  test("shows searching state once a search runs past the reveal delay", async () => {
    const wrapper = createStatus({ active: true });
    vi.advanceTimersByTime(400);
    await nextTick();
    expect(wrapper.find('[data-state="searching"]').exists()).toBe(true);
    expect(wrapper.findComponent(IndeterminateBar).exists()).toBe(true);
  });

  test("reveals the searching bar even when progress events arrive faster than the delay", async () => {
    const wrapper = createStatus({ active: true });
    // a slow search emits a progress event each window; the reveal delay must
    // measure from when the search started, not restart on every event
    for (let i = 0; i < 5; i++) {
      vi.advanceTimersByTime(100);
      await wrapper.setProps({ status: { active: true, done: false, matches: i, scannedTo: `t${i}` } });
      await nextTick();
    }
    expect(wrapper.find('[data-state="searching"]').exists()).toBe(true);
  });

  test("shows the empty state when a search finishes with no matches", async () => {
    const wrapper = createStatus({ active: false, done: true, matches: 0, reason: "exhausted" });
    await nextTick();
    expect(wrapper.find('[data-state="empty"]').exists()).toBe(true);
  });

  test("shows a completion summary for a slow exhausted search", async () => {
    const wrapper = createStatus({ active: true });
    vi.advanceTimersByTime(400);
    await nextTick();
    await wrapper.setProps({ status: { active: false, done: true, matches: 3, reason: "exhausted" } });
    await nextTick();
    expect(wrapper.find('[data-state="exhausted"]').exists()).toBe(true);
    expect(wrapper.text()).toContain("3");
  });

  test("shows a capped summary for a slow capped search", async () => {
    const wrapper = createStatus({ active: true });
    vi.advanceTimersByTime(400);
    await nextTick();
    await wrapper.setProps({
      status: { active: false, done: true, matches: 50, reason: "capped", scannedTo: "2026-06-01T13:10:00Z" },
    });
    await nextTick();
    expect(wrapper.find('[data-state="capped"]').exists()).toBe(true);
  });

  test("stays quiet for a fast search that returned matches", async () => {
    const wrapper = createStatus({ active: true });
    vi.advanceTimersByTime(100);
    await nextTick();
    await wrapper.setProps({ status: { active: false, done: true, matches: 5, reason: "capped" } });
    await nextTick();
    expect(wrapper.find("[data-state]").exists()).toBe(false);
  });
});
