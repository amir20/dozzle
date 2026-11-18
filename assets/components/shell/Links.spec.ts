/**
 * @vitest-environment jsdom
 */
import { mount, RouterLinkStub } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { ref } from "vue";
import { createI18n } from "vue-i18n";
import Links from "./Links.vue";

const linked = ref(false);

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { enableNotifications: true, hosts: [] },
  withBase: (p: string) => p,
}));
vi.mock("@/composable/cloud/cloudSurface", () => ({ useCloudSurface: () => ({ linked }) }));
vi.mock("@/composable/cloud/recentAlerts", () => ({
  useRecentAlerts: () => ({ unseen: ref(false), fetchRecentAlerts: vi.fn(), refreshRecentAlerts: vi.fn() }),
}));

const i18n = createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false, messages: { en: {} } });
const mountLinks = () =>
  mount(Links, {
    global: { plugins: [i18n], stubs: { RouterLink: RouterLinkStub, Announcements: true, Dropdown: true } },
  });

describe("<Links /> bell", () => {
  // What fired lives in the cloud drawer; the bell is always the rules and
  // destinations, linked or not.
  test.each([false, true])("goes to the notifications page (cloud linked: %s)", (isLinked) => {
    linked.value = isLinked;
    const bell = mountLinks().findComponent(RouterLinkStub);
    expect(bell.attributes("data-testid")).toBe("notifications");
    expect(bell.props("to")).toEqual({ name: "/notifications" });
  });
});
