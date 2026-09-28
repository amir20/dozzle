/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { createI18n } from "vue-i18n";
import SetupLoginStep from "./SetupLoginStep.vue";
import type { SetupStatus } from "@/composable/setup/setup";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [] },
  withBase: (path: string) => path,
}));

const i18n = createI18n({
  legacy: false,
  locale: "en",
  missingWarn: false,
  fallbackWarn: false,
  messages: { en: { setup: { login: { mismatch: "Passwords don't match." } } } },
});

function mountStep() {
  const status = { dataPersisted: true, canWrite: true, authProvider: "none", pending: {}, locked: {} } as SetupStatus;
  return mount(SetupLoginStep, { props: { status }, global: { plugins: [i18n], stubs: { SetupRestarting: true } } });
}

describe("SetupLoginStep password confirmation", () => {
  test("stays quiet while the confirmation is still being typed", async () => {
    const wrapper = mountStep();
    const [password, confirm] = wrapper.findAll('input[type="password"]');
    await password.setValue("correct-horse");
    await confirm.setValue("correct");
    expect(wrapper.text()).not.toContain("Passwords don't match.");
  });

  test("says so once the passwords differ", async () => {
    const wrapper = mountStep();
    const [password, confirm] = wrapper.findAll('input[type="password"]');
    await password.setValue("correct-horse");
    await confirm.setValue("correct-hoarse");
    expect(wrapper.text()).toContain("Passwords don't match.");
  });
});
