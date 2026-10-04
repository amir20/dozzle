/**
 * @vitest-environment jsdom
 */
import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { createI18n } from "vue-i18n";

vi.mock("@/stores/config", () => ({
  default: { base: "", mode: "server" },
  withBase: (path: string) => path,
}));

import SetupTogglesForm from "./SetupTogglesForm.vue";
import type { SetupStatus } from "@/composable/setup/setup";

const i18n = createI18n({
  legacy: false,
  locale: "en",
  missingWarn: false,
  fallbackWarn: false,
  messages: { en: { setup: { actions: { locked: "Set by {env}" } }, settings: { "after-restart": "After restart" } } },
});

function status(over: Partial<SetupStatus> = {}): SetupStatus {
  return {
    mode: "server",
    dataPersisted: true,
    authProvider: "simple",
    usersFileExists: true,
    enableActions: false,
    enableShell: false,
    locked: { authProvider: false, enableActions: false, enableShell: false },
    pending: {},
    canRestart: true,
    windowOpen: false,
    canWrite: true,
    ...over,
  };
}

let calls: { url: string; init?: RequestInit }[];
let patchStatus = 204;

beforeEach(() => {
  calls = [];
  patchStatus = 204;
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url: string, init?: RequestInit) => {
      calls.push({ url, init });
      if (url === "/api/setup") return new Response(JSON.stringify(status({ pending: { enableActions: true } })));
      return new Response(null, { status: init?.method === "PATCH" ? patchStatus : 200 });
    }),
  );
});

afterEach(() => vi.unstubAllGlobals());

const mountForm = (s: SetupStatus, autosave = false) =>
  mount(SetupTogglesForm, { props: { status: s, autosave }, global: { plugins: [i18n] } });

const patches = () => calls.filter((c) => c.init?.method === "PATCH").map((c) => JSON.parse(String(c.init?.body)));

describe("SetupTogglesForm", () => {
  test("in Settings a switch saves the moment it changes", async () => {
    const wrapper = mountForm(status(), true);
    await wrapper.findAll("input[type=checkbox]")[0].setValue(true);
    await flushPromises();
    expect(patches()).toEqual([{ enableActions: true }]);
  });

  test("in the wizard nothing is saved until Next", async () => {
    const wrapper = mountForm(status());
    await wrapper.findAll("input[type=checkbox]")[1].setValue(true);
    await flushPromises();
    expect(patches()).toEqual([]);
    expect((wrapper.vm as unknown as { dirty: boolean }).dirty).toBe(true);
    await (wrapper.vm as unknown as { save: () => Promise<boolean> }).save();
    expect(patches()).toEqual([{ enableShell: true }]);
  });

  test("a switch fixed by a variable is locked and names it", () => {
    const wrapper = mountForm(status({ locked: { authProvider: false, enableActions: true, enableShell: false } }));
    // Only shell is still a switch; the locked one reads as text.
    expect(wrapper.findAll("input[type=checkbox]")).toHaveLength(1);
    expect(wrapper.text()).toContain("Set by DOZZLE_ENABLE_ACTIONS");
  });

  test("an account without every role sees the values as text", () => {
    const wrapper = mountForm(status({ canWrite: false }), true);
    expect(wrapper.find("input[type=checkbox]").exists()).toBe(false);
  });

  test("a saved switch waiting for a restart says so in Settings", () => {
    const wrapper = mountForm(status({ pending: { enableShell: true } }), true);
    expect(wrapper.text()).toContain("After restart");
    expect((wrapper.findAll("input[type=checkbox]")[1].element as HTMLInputElement).checked).toBe(true);
  });

  test("a failed save puts the switch back", async () => {
    patchStatus = 500;
    const wrapper = mountForm(status(), true);
    const actions = wrapper.findAll("input[type=checkbox]")[0];
    await actions.setValue(true);
    await flushPromises();
    expect((actions.element as HTMLInputElement).checked).toBe(false);
    expect(patches()).toHaveLength(1);
  });
});
