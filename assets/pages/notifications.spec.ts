/**
 * @vitest-environment jsdom
 */
import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, describe, expect, test, vi } from "vitest";
import { reactive, ref } from "vue";
import { createI18n } from "vue-i18n";

const route = reactive({ query: {} as Record<string, string | undefined>, hash: "" });
const replace = vi.fn((to: { query: Record<string, string | undefined> }) => (route.query = to.query));
vi.mock("vue-router", () => ({ useRoute: () => route, useRouter: () => ({ replace }) }));
vi.mock("@/stores/config", () => ({ __esModule: true, default: { hosts: [] }, withBase: (p: string) => p }));
const linked = ref(true);
vi.mock("@/composable/cloud/cloudSurface", () => ({ useCloudSurface: () => ({ linked }) }));
const openRail = vi.fn();
vi.mock("@/composable/cloud/cloudRail", () => ({ useCloudRail: () => ({ openRail }) }));
vi.mock("@/composable/cloud/recentAlerts", () => ({ useRecentAlerts: () => ({}) }));
// The page frame drags in the whole header; only the page body is under test.
vi.mock("@/components/shell/PageWithLinks.vue", async () => {
  const { defineComponent, h } = await import("vue");
  return {
    default: defineComponent(
      (_, { slots }) =>
        () =>
          h("div", slots.default?.()),
    ),
  };
});

import Notifications from "./notifications.vue";

const i18n = createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false, messages: { en: {} } });

function mountPage(query: Record<string, string>) {
  route.query = query;
  vi.stubGlobal(
    "fetch",
    vi.fn(() => Promise.resolve(new Response("[]"))),
  );
  return mount(Notifications, {
    global: { plugins: [i18n], stubs: { EmptyState: true } },
  });
}

afterEach(() => {
  vi.unstubAllGlobals();
  openRail.mockReset();
  replace.mockClear();
});

describe("/notifications", () => {
  test("?tab=activity opens the drawer on Alerts and leaves the rules on the page", async () => {
    const wrapper = mountPage({ tab: "activity" });
    await flushPromises();
    expect(openRail).toHaveBeenCalledWith("alerts");
    expect(route.query.tab).toBeUndefined();
    // Two tabs, the rules and the destinations: the history is not one of them.
    expect(wrapper.findAll('[role="tab"]')).toHaveLength(2);
  });

  test("unlinked, ?tab=activity just lands on the rules", async () => {
    linked.value = false;
    mountPage({ tab: "activity" });
    await flushPromises();
    expect(openRail).not.toHaveBeenCalled();
    expect(route.query.tab).toBeUndefined();
    linked.value = true;
  });
});
