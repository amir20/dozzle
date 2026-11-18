/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { nextTick, ref } from "vue";
import { createI18n } from "vue-i18n";
import CloudDrawer from "./CloudDrawer.vue";

const panel = ref<"chat" | "metrics" | "alerts" | undefined>("alerts");
const scoped = ref(true);
const openRail = vi.fn((id: "chat" | "metrics" | "alerts") => (panel.value = id));

vi.mock("@/composable/cloud/cloudRail", () => ({
  useCloudRail: () => ({ panel, openRail, closeRail: vi.fn() }),
  useViewAlerts: () => ({ unseen: ref(false), scoped }),
}));
vi.mock("@/composable/cloud/cloudChat", () => ({ useCloudChat: () => ({ messages: ref([]), reset: vi.fn() }) }));

const i18n = createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false, messages: { en: {} } });

describe("<CloudDrawer />", () => {
  test("off a log view the alerts tab is gone, the chat takes over, and metrics are per host", async () => {
    const wrapper = mount(CloudDrawer, {
      global: {
        plugins: [i18n],
        stubs: { ChatPane: true, RailMetrics: true, RailHostMetrics: true, RailAlerts: true },
      },
    });
    expect(wrapper.findAll("nav button[aria-pressed]")).toHaveLength(3);

    scoped.value = false;
    await nextTick();
    expect(wrapper.findAll("nav button[aria-pressed]")).toHaveLength(2);
    expect(panel.value).toBe("chat");

    panel.value = "metrics";
    await nextTick();
    expect(wrapper.findComponent({ name: "RailHostMetrics" }).exists()).toBe(true);
    expect(wrapper.findComponent({ name: "RailMetrics" }).exists()).toBe(false);
  });
});
