/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { afterEach, describe, expect, test, vi } from "vitest";
import { ref } from "vue";
import { createI18n } from "vue-i18n";
import CloudMark from "./CloudMark.vue";
import { isMobile } from "@/composable/ui/media";

const mobile = isMobile as unknown as { value: boolean };
const panel = ref<string>();
const openRail = vi.fn((id: string) => (panel.value = id));
const toggleRail = vi.fn((id: string) => (panel.value = panel.value === id ? undefined : id));

vi.mock("@/composable/ui/media", async () => {
  const { ref } = await import("vue");
  return { isMobile: ref(false) };
});
vi.mock("@/composable/cloud/cloudRail", () => ({ useCloudRail: () => ({ panel, openRail, toggleRail }) }));
vi.mock("@/composable/cloud/cloudConfig", () => ({
  useCloudConfig: () => ({ cloudStatusError: ref(false), ensureCloudStatus: vi.fn() }),
}));

const i18n = createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false, messages: { en: {} } });
const mountMark = () => mount(CloudMark, { global: { plugins: [i18n] } }).find("button");

afterEach(() => {
  panel.value = undefined;
  mobile.value = false;
  openRail.mockClear();
  toggleRail.mockClear();
});

describe("<CloudMark />", () => {
  test("on the rail it toggles the account panel and shows it is open", async () => {
    const wrapper = mountMark();
    await wrapper.trigger("click");
    expect(toggleRail).toHaveBeenCalledWith("account");
    expect(wrapper.attributes("aria-pressed")).toBe("true");
    expect(wrapper.classes()).toContain("bg-base-300");
  });

  // On a phone the mark sits in the sheet's header, so a second tap stays put.
  test("on a phone it opens the account", async () => {
    mobile.value = true;
    const wrapper = mountMark();
    await wrapper.trigger("click");
    expect(openRail).toHaveBeenCalledWith("account");
    expect(toggleRail).not.toHaveBeenCalled();
  });
});
