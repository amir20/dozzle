/**
 * @vitest-environment jsdom
 */
import { mount, RouterLinkStub } from "@vue/test-utils";
import { afterEach, describe, expect, test, vi } from "vitest";
import { ref } from "vue";
import { createI18n } from "vue-i18n";
import MobileTabBar from "./MobileTabBar.vue";

const linked = ref(false);
const unseen = ref(false);
const openRail = vi.fn();
const widen = vi.fn();

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { enableNotifications: true, hosts: [] },
  withBase: (p: string) => p,
}));
vi.mock("vue-router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("vue-router")>()),
  useRoute: () => ({ fullPath: "/", name: "/" }),
}));
vi.mock("@/composable/cloud/cloudSurface", () => ({ useCloudSurface: () => ({ linked }) }));
vi.mock("@/composable/cloud/cloudRail", () => ({ useCloudRail: () => ({ openRail }) }));
vi.mock("@/composable/cloud/cloudScope", () => ({ useCloudScope: () => ({ widen }) }));
vi.mock("@/composable/cloud/recentAlerts", () => ({ useRecentAlerts: () => ({ unseen }) }));
vi.mock("@/composable/ui/compactOnScroll", () => ({ useCompactOnScroll: () => ref(false) }));

const i18n = createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false, messages: { en: {} } });
const mountBar = () =>
  mount(MobileTabBar, {
    global: {
      plugins: [i18n],
      stubs: { RouterLink: RouterLinkStub, SideMenu: true, BrandMark: true, ProBadge: true },
    },
  });

afterEach(() => {
  linked.value = false;
  unseen.value = false;
  openRail.mockClear();
  widen.mockClear();
});

describe("<MobileTabBar /> notifications", () => {
  test("goes to the rules page when nothing unseen fired", () => {
    linked.value = true;
    const tab = mountBar().findComponent<typeof RouterLinkStub>('[data-testid="notifications-tab"]');
    expect(tab.props("to")).toEqual({ name: "/notifications" });
  });

  test("goes to the rules page without a cloud link, dot or not", () => {
    unseen.value = true;
    const tab = mountBar().findComponent<typeof RouterLinkStub>('[data-testid="notifications-tab"]');
    expect(tab.props("to")).toEqual({ name: "/notifications" });
  });

  test("opens the cloud sheet on every container's alerts when something unseen fired", async () => {
    linked.value = true;
    unseen.value = true;
    const tab = mountBar().find('[data-testid="notifications-tab"]');
    expect(tab.element.tagName).toBe("BUTTON");
    await tab.trigger("click");
    expect(widen).toHaveBeenCalledWith("bell");
    expect(openRail).toHaveBeenCalledWith("alerts");
  });
});
