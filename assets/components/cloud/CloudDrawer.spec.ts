/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { nextTick, ref } from "vue";
import { createI18n } from "vue-i18n";
import CloudDrawer from "./CloudDrawer.vue";

const panel = ref<"chat" | "metrics" | "alerts" | undefined>("metrics");
const scoped = ref(true);
const openRail = vi.fn((id: "chat" | "metrics" | "alerts") => (panel.value = id));

vi.mock("@/composable/cloud/cloudRail", () => ({
  useCloudRail: () => ({ panel, openRail, closeRail: vi.fn() }),
  useViewAlerts: () => ({ unseen: ref(false), scoped }),
}));
vi.mock("@/composable/cloud/cloudChat", () => ({ useCloudChat: () => ({ messages: ref([]), reset: vi.fn() }) }));

const i18n = createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false, messages: { en: {} } });

describe("<CloudDrawer />", () => {
  test("off a log view only the chat is offered, and an open tab falls back to it", async () => {
    const wrapper = mount(CloudDrawer, {
      global: { plugins: [i18n], stubs: { ChatPane: true, RailMetrics: true, RailAlerts: true } },
    });
    expect(wrapper.findAll("nav button[aria-pressed]")).toHaveLength(3);

    scoped.value = false;
    await nextTick();
    expect(wrapper.findAll("nav button[aria-pressed]")).toHaveLength(1);
    expect(panel.value).toBe("chat");
  });
});
