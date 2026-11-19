/**
 * @vitest-environment jsdom
 */
import { mount, RouterLinkStub } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { ref } from "vue";
import { createI18n } from "vue-i18n";
import MobileTabBar from "./MobileTabBar.vue";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { enableNotifications: true, hosts: [] },
  withBase: (p: string) => p,
}));
vi.mock("vue-router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("vue-router")>()),
  useRoute: () => ({ fullPath: "/", name: "/" }),
}));
vi.mock("@/composable/ui/compactOnScroll", () => ({ useCompactOnScroll: () => ref(false) }));

const i18n = createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false, messages: { en: {} } });
const mountBar = () =>
  mount(MobileTabBar, {
    global: {
      plugins: [i18n],
      stubs: { RouterLink: RouterLinkStub, SideMenu: true, BrandMark: true, ProBadge: true },
    },
  });

describe("<MobileTabBar /> notifications", () => {
  // What fired lives in the cloud sheet; the tab is always the rules and destinations.
  test("goes to the notifications page", () => {
    const tab = mountBar().findComponent<typeof RouterLinkStub>('[data-testid="notifications-tab"]');
    expect(tab.props("to")).toEqual({ name: "/notifications" });
  });
});
