/**
 * @vitest-environment jsdom
 */
import { mount, RouterLinkStub } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { ref } from "vue";
import { createI18n } from "vue-i18n";
import Links from "./Links.vue";

const linked = ref(false);
const openRail = vi.fn();

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { enableNotifications: true, hosts: [] },
  withBase: (p: string) => p,
}));
vi.mock("@/composable/cloud/cloudSurface", () => ({ useCloudSurface: () => ({ linked }) }));
vi.mock("@/composable/cloud/cloudRail", () => ({ useCloudRail: () => ({ openRail }) }));
vi.mock("@/composable/cloud/recentAlerts", () => ({
  useRecentAlerts: () => ({ unseen: ref(false), fetchRecentAlerts: vi.fn(), refreshRecentAlerts: vi.fn() }),
}));

const i18n = createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false, messages: { en: {} } });
const mountLinks = () =>
  mount(Links, {
    global: { plugins: [i18n], stubs: { RouterLink: RouterLinkStub, Announcements: true, Dropdown: true } },
  });

describe("<Links /> bell", () => {
  test("goes to the rules page while cloud is not linked", () => {
    linked.value = false;
    const bell = mountLinks().find('[data-testid="notifications"]');
    expect(bell.element.tagName).not.toBe("BUTTON");
  });

  test("opens the drawer's Alerts panel once cloud is linked", async () => {
    linked.value = true;
    const bell = mountLinks().find('[data-testid="notifications"]');
    expect(bell.element.tagName).toBe("BUTTON");
    await bell.trigger("click");
    expect(openRail).toHaveBeenCalledWith("alerts");
  });
});
