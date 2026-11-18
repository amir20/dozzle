/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { computed, ref } from "vue";
import { createI18n } from "vue-i18n";
import RailAlerts from "./RailAlerts.vue";

vi.mock("@/stores/config", () => ({ __esModule: true, default: { enableNotifications: true, hosts: [] } }));
vi.mock("@/composable/cloud/recentAlerts", () => ({
  useRecentAlerts: () => ({
    fetchRecentAlerts: vi.fn(),
    refreshRecentAlerts: vi.fn(),
    markAlertsSeen: vi.fn(),
    loading: ref(false),
    loaded: ref(true),
    failed: ref(false),
  }),
}));
const closeRail = vi.fn();
vi.mock("@/composable/cloud/cloudRail", () => ({
  useCloudRail: () => ({ closeRail }),
  useViewAlerts: () => ({ visible: computed(() => []), scoped: computed(() => true), single: computed(() => true) }),
}));
vi.mock("@/composable/cloud/cloudConfig", () => ({
  useCloudConfig: () => ({
    cloudStatus: ref({ plan: { retention_days: 30 } }),
    ensureCloudStatus: vi.fn(),
  }),
}));

const i18n = createI18n({
  legacy: false,
  locale: "en",
  missingWarn: false,
  fallbackWarn: false,
  messages: {
    en: {
      "cloud-rail": {
        "alerts-empty-here": "Nothing has fired on what you're looking at.",
        "alerts-retention":
          "Dozzle Cloud keeps {n} day of alert history. | Dozzle Cloud keeps {n} days of alert history.",
        "create-alert": "Create an alert rule",
      },
    },
  },
});

const RouterLinkStub = { props: ["to"], template: "<a data-link><slot /></a>" };

describe("<RailAlerts />", () => {
  // Nothing in scope still says how far back that reaches and where a rule is made.
  test("an empty scope states the history kept and links to a new rule", async () => {
    const wrapper = mount(RailAlerts, { global: { plugins: [i18n], stubs: { RouterLink: RouterLinkStub } } });
    const empty = wrapper.find("[data-empty]");
    expect(empty.text()).toContain("Nothing has fired on what you're looking at.");
    expect(empty.text()).toContain("Dozzle Cloud keeps 30 days of alert history.");

    const create = empty.findComponent(RouterLinkStub);
    expect(create.text()).toBe("Create an alert rule");
    expect(create.props("to")).toEqual({ name: "/notifications", query: { action: "create-alert" } });
    await create.trigger("click");
    expect(closeRail).toHaveBeenCalled();
  });
});
