/**
 * @vitest-environment jsdom
 */
import { mount, RouterLinkStub } from "@vue/test-utils";
import { afterEach, describe, expect, test, vi } from "vitest";
import { computed, ref, toValue, type MaybeRefOrGetter } from "vue";
import { createI18n } from "vue-i18n";
import CloudRail from "./CloudRail.vue";

const available = ref(false);
const canLink = ref(true);
const enableNotifications = ref(true);
const railDots: MaybeRefOrGetter<boolean>[] = [];

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: {
    get enableNotifications() {
      return enableNotifications.value;
    },
    hosts: [],
  },
}));
vi.mock("@/composable/cloud/cloudSurface", () => ({ useCloudSurface: () => ({ canLink }) }));
vi.mock("@/composable/cloud/cloudRail", () => ({
  RAIL_WIDTH: 48,
  useCloudRail: () => ({ panel: ref(), available, toggleRail: vi.fn(), hideRail: vi.fn() }),
  useRailItems: (dots: MaybeRefOrGetter<boolean>) => {
    railDots.push(dots);
    return computed(() => [
      { id: "findings", label: "Findings", icon: "i", dot: false },
      { id: "alerts", label: "Alerts", icon: "i", dot: false },
    ]);
  },
}));

const i18n = createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false, messages: { en: {} } });
const mountRail = () =>
  mount(CloudRail, {
    global: {
      plugins: [i18n],
      stubs: {
        RouterLink: RouterLinkStub,
        CloudMark: { template: "<div data-cloud-mark />" },
        CloudLinkPopover: { template: "<div data-cloud-link />" },
        Announcements: { template: "<div data-announcements />" },
        UserMenu: { template: "<div data-user-menu />" },
      },
    },
  });

afterEach(() => {
  railDots.length = 0;
  available.value = false;
  canLink.value = true;
  enableNotifications.value = true;
});

describe("<CloudRail /> app nav", () => {
  // The foot of the strip is the app's own nav, so it is the same linked or not.
  test.each([false, true])("carries what's new, the rules, settings and the account (linked: %s)", (linked) => {
    available.value = linked;
    const rail = mountRail();
    const links = rail.findAllComponents(RouterLinkStub);
    expect(links.map((l) => l.attributes("data-testid"))).toEqual(["notifications", "settings"]);
    expect(links[0].props("to")).toEqual({ name: "/notifications" });
    expect(links[1].props("to")).toEqual({ name: "/settings" });
    expect(rail.find("[data-announcements]").exists()).toBe(true);
    expect(rail.find("[data-user-menu]").exists()).toBe(true);
  });

  test("leaves the rules out when notifications are off", () => {
    enableNotifications.value = false;
    const links = mountRail().findAllComponents(RouterLinkStub);
    expect(links.map((l) => l.attributes("data-testid"))).toEqual(["settings"]);
  });
});

describe("<CloudRail /> cloud half", () => {
  test("linked: the cloud mark and the panels", () => {
    available.value = true;
    const rail = mountRail();
    expect(rail.find("[data-cloud-mark]").exists()).toBe(true);
    expect(rail.find("[data-cloud-link]").exists()).toBe(false);
    expect(rail.findAll("button[aria-pressed]")).toHaveLength(2);
  });

  test("not linked: only the way in, and no panels", () => {
    const rail = mountRail();
    expect(rail.find("[data-cloud-mark]").exists()).toBe(false);
    expect(rail.find("[data-cloud-link]").exists()).toBe(true);
    expect(rail.findAll("button[aria-pressed]")).toHaveLength(0);
  });

  test("not linked and not allowed to link: no cloud at all", () => {
    canLink.value = false;
    const rail = mountRail();
    expect(rail.find("[data-cloud-mark]").exists()).toBe(false);
    expect(rail.find("[data-cloud-link]").exists()).toBe(false);
  });

  // Unlinked, the findings dot has nothing to sit on, and asking for it is a
  // 503 from the server on every page.
  test.each([false, true])("asks Cloud for the findings dot only when linked (linked: %s)", (linked) => {
    available.value = linked;
    mountRail();
    expect(railDots.map((d) => toValue(d))).toEqual([linked]);
  });
});
