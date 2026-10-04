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

import SetupLoginForm from "./SetupLoginForm.vue";
import type { SetupStatus } from "@/composable/setup/setup";

const i18n = createI18n({
  legacy: false,
  locale: "en",
  missingWarn: false,
  fallbackWarn: false,
  messages: {
    en: {
      setup: { actions: { locked: "Set by {env}" }, login: { "create-account": "Create account" } },
      settings: { "login-off": "No login", "login-docs": "Authentication guide" },
    },
  },
});

function status(over: Partial<SetupStatus> = {}): SetupStatus {
  return {
    mode: "server",
    dataPersisted: true,
    authProvider: "none",
    usersFileExists: false,
    enableActions: false,
    enableShell: false,
    locked: { authProvider: false, enableActions: false, enableShell: false },
    pending: {},
    canRestart: true,
    windowOpen: true,
    canWrite: true,
    ...over,
  };
}

let calls: { url: string; init?: RequestInit }[];

beforeEach(() => {
  calls = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url: string, init?: RequestInit) => {
      calls.push({ url, init });
      if (url === "/api/setup") return new Response(JSON.stringify(status({ pending: { authProvider: "simple" } })));
      return new Response(null, { status: 204 });
    }),
  );
});

afterEach(() => vi.unstubAllGlobals());

const mountForm = (s: SetupStatus, standalone = true) =>
  mount(SetupLoginForm, { props: { status: s, standalone }, global: { plugins: [i18n] } });

describe("SetupLoginForm in Settings", () => {
  test("offers the wizard's three ways in while there is no login", () => {
    const wrapper = mountForm(status());
    expect(wrapper.findAll("[role=tab]")).toHaveLength(3);
    expect(wrapper.findAll("input[type=password]")).toHaveLength(2);
  });

  // The restart banner above owns the one primary button while it shows.
  test("the save button is plain while a restart is offered", () => {
    const button = (s: SetupStatus) =>
      mountForm(s)
        .findAll("button")
        .find((b) => b.text() === "Create account")!;
    expect(button(status()).classes()).toContain("btn-primary");
    expect(button(status({ pending: { enableActions: true } })).classes()).not.toContain("btn-primary");
  });

  test("creates the account with its own button", async () => {
    const wrapper = mountForm(status());
    await wrapper.find("input[autocomplete=username]").setValue("admin");
    const [password, confirm] = wrapper.findAll("input[type=password]");
    await password.setValue("correct-horse");
    await confirm.setValue("correct-horse");
    const button = wrapper.findAll("button").find((b) => b.text() === "Create account")!;
    expect(button.attributes("disabled")).toBeUndefined();
    await button.trigger("click");
    await flushPromises();
    const post = calls.find((c) => c.url === "/api/setup/account");
    expect(JSON.parse(String(post?.init?.body))).toEqual({ username: "admin", password: "correct-horse" });
  });

  test("the wizard drives saving from its footer, so there is no button of its own", () => {
    const wrapper = mountForm(status(), false);
    expect(wrapper.findAll("button").some((b) => b.text() === "Create account")).toBe(false);
  });

  test("once login is on it shows the provider and the guide, not a form", () => {
    const wrapper = mountForm(status({ authProvider: "simple" }));
    expect(wrapper.text()).toContain("auth: simple");
    expect(wrapper.find("a[href='https://dozzle.dev/guide/authentication']").exists()).toBe(true);
    expect(wrapper.find("input").exists()).toBe(false);
  });

  test("a provider fixed by a variable says which one", () => {
    const wrapper = mountForm(status({ locked: { authProvider: true, enableActions: false, enableShell: false } }));
    expect(wrapper.text()).toContain("No login");
    expect(wrapper.text()).toContain("Set by DOZZLE_AUTH_PROVIDER");
    expect(wrapper.find("input").exists()).toBe(false);
  });

  test("with the window closed it reports instead of offering a form that would be refused", () => {
    const wrapper = mountForm(status({ canWrite: false, windowOpen: false }));
    expect(wrapper.text()).toContain("No login");
    expect(wrapper.find("input").exists()).toBe(false);
  });
});
