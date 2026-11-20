/**
 * @vitest-environment jsdom
 */
import { mount, RouterLinkStub } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { ref } from "vue";
import { createI18n } from "vue-i18n";
import Links from "./Links.vue";

const linked = ref(false);
const mobile = ref(false);
const railAvailable = ref(false);
const unseen = ref(false);
const openCloud = vi.fn();

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { enableNotifications: true, hosts: [] },
  withBase: (p: string) => p,
}));
vi.mock("@/composable/cloud/cloudSurface", () => ({ useCloudSurface: () => ({ linked }) }));
vi.mock("@/composable/cloud/recentAlerts", () => ({
  useRecentAlerts: () => ({ unseen, fetchRecentAlerts: vi.fn(), refreshRecentAlerts: vi.fn() }),
}));
vi.mock("@/composable/cloud/cloudRail", () => ({ useCloudRail: () => ({ available: railAvailable, openCloud }) }));
vi.mock("@/composable/ui/media", () => ({
  get isMobile() {
    return mobile;
  },
}));

const i18n = createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false, messages: { en: {} } });
const mountLinks = () =>
  mount(Links, {
    global: { plugins: [i18n], stubs: { RouterLink: RouterLinkStub, Announcements: true, Dropdown: true } },
  });

// A phone has no rail, so the header is the way into Cloud from any page.
describe("<Links /> phone cloud button", () => {
  test("shows on a phone once linked, and opens on unseen alerts", async () => {
    mobile.value = true;
    railAvailable.value = true;
    unseen.value = true;
    const button = mountLinks().find('[data-testid="cloud-open"]');
    expect(button.exists()).toBe(true);
    await button.trigger("click");
    expect(openCloud).toHaveBeenCalledWith(true);
  });

  test.each([
    { mobile: false, available: true },
    { mobile: true, available: false },
  ])("is absent when mobile=$mobile and linked=$available", ({ mobile: m, available }) => {
    mobile.value = m;
    railAvailable.value = available;
    expect(mountLinks().find('[data-testid="cloud-open"]').exists()).toBe(false);
  });
});
