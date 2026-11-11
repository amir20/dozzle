// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";
import { createI18n } from "vue-i18n";
import type { NotificationRule } from "@/types/notifications";
import { LOG_COOLDOWN_STOPS } from "@/composable/notifications/alertForm";
import LogAlertFields from "./LogAlertFields.vue";

vi.mock("@/stores/config", () => ({
  default: { base: "", hosts: [] },
  withBase: (path: string) => path,
}));

afterEach(() => vi.useRealTimers());

function mountFields(
  options: {
    alert?: NotificationRule;
    prefill?: { logExpression?: string; cooldown?: number; immediate?: boolean };
    immediate?: boolean;
  } = {},
) {
  vi.useFakeTimers();
  return mount(LogAlertFields, {
    props: {
      immediate: options.alert?.immediate ?? options.prefill?.immediate ?? false,
      containerExpression: "true",
      hasContainers: false,
      isLoading: false,
      validatePreview: vi.fn().mockResolvedValue({ data: null }),
      ...options,
    },
    global: {
      plugins: [createI18n({ legacy: false, missingWarn: false, fallbackWarn: false })],
      stubs: { ExpressionField: true, LogList: true },
    },
  });
}

describe("log alert cooldown", () => {
  it("keeps Cloud's 10-second default when send immediately is enabled", async () => {
    const wrapper = mountFields();
    expect(wrapper.vm.typeFields.cooldown).toBe(0);
    await wrapper.setProps({ immediate: true });
    expect(wrapper.vm.typeFields.cooldown).toBe(10);
    expect(wrapper.get("input[type=range]").attributes("aria-valuetext")).toBe("10s");
    await wrapper.setProps({ immediate: false });
    expect(wrapper.vm.typeFields.cooldown).toBe(0);
    wrapper.unmount();
  });

  it("defaults a new immediate rule to 10 seconds on mount", () => {
    const wrapper = mountFields({ immediate: true });
    expect(wrapper.vm.typeFields.cooldown).toBe(10);
    wrapper.unmount();
  });

  it.each([0, undefined])("preserves a saved immediate rule with cooldown=%s", async (cooldown) => {
    const wrapper = mountFields({
      alert: { logExpression: "true", cooldown, immediate: true } as NotificationRule,
    });
    expect(wrapper.vm.typeFields.cooldown).toBe(0);
    await wrapper.setProps({ immediate: false });
    await wrapper.setProps({ immediate: true });
    expect(wrapper.vm.typeFields.cooldown).toBe(0);
    wrapper.unmount();
  });

  it("keeps each delivery mode's chosen cooldown when switching modes", async () => {
    const wrapper = mountFields({ prefill: { logExpression: "true", cooldown: 30 } });
    await wrapper.setProps({ immediate: true });
    expect(wrapper.vm.typeFields.cooldown).toBe(10);
    await wrapper.get("input[type=range]").setValue(0);
    await wrapper.setProps({ immediate: false });
    expect(wrapper.vm.typeFields.cooldown).toBe(30);
    await wrapper.setProps({ immediate: true });
    expect(wrapper.vm.typeFields.cooldown).toBe(0);
    wrapper.unmount();
  });

  it("is available on an ordinary log rule and defaults to off", async () => {
    const wrapper = mountFields({ prefill: { logExpression: 'message contains "429"' } });
    const range = wrapper.get("input[type=range]");
    expect(wrapper.vm.typeFields.cooldown).toBe(0);
    expect(wrapper.text()).toContain("notifications.alert-form.no-cooldown");

    await range.setValue(LOG_COOLDOWN_STOPS.indexOf(60));
    expect(wrapper.vm.typeFields).toMatchObject({ logExpression: 'message contains "429"', cooldown: 60 });

    await range.setValue(0);
    expect(wrapper.vm.typeFields.cooldown).toBe(0);
    wrapper.unmount();
  });

  it.each([false, true])("preserves a saved cooldown with immediate=%s", (immediate) => {
    const wrapper = mountFields({
      alert: { logExpression: "true", cooldown: 45, immediate } as NotificationRule,
    });
    expect(wrapper.vm.typeFields.cooldown).toBe(45);
    expect(wrapper.get("input[type=range]").attributes("aria-valuetext")).toBe("45s");
    wrapper.unmount();
  });

  it("preserves a prefilled cooldown when returning from destination setup", () => {
    const wrapper = mountFields({ prefill: { logExpression: "true", cooldown: 30 } });
    expect(wrapper.vm.typeFields.cooldown).toBe(30);
    wrapper.unmount();
  });

  it("preserves an immediate prefilled cooldown when returning from destination setup", () => {
    const wrapper = mountFields({ prefill: { logExpression: "true", cooldown: 60, immediate: true } });
    expect(wrapper.vm.typeFields.cooldown).toBe(60);
    wrapper.unmount();
  });
});
