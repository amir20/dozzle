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

vi.mock("vue-router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("vue-router")>()),
  useRoute: () => ({ fullPath: "/settings/security" }),
}));

import SetupRestartBanner from "./SetupRestartBanner.vue";
import type { SetupStatus } from "@/composable/setup/setup";

const i18n = createI18n({
  legacy: false,
  locale: "en",
  missingWarn: false,
  fallbackWarn: false,
  messages: {
    en: {
      settings: { "restart-pending": "1 change applies after a restart | {count} changes apply after a restart" },
      setup: {
        restart: { button: "Restart Dozzle", manual: "Dozzle can't restart itself here." },
        actions: { "window-closed": "Setup window closed.", "no-access": "Your account can't change these settings." },
      },
    },
  },
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
    pending: { enableActions: true, enableShell: true },
    canRestart: true,
    windowOpen: false,
    canWrite: true,
    ...over,
  };
}

let calls: string[];

beforeEach(() => {
  calls = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url: string, init?: RequestInit) => {
      calls.push(`${init?.method ?? "GET"} ${url}`);
      return new Response(null, { status: 202 });
    }),
  );
});

afterEach(() => vi.unstubAllGlobals());

const mountBanner = (s: SetupStatus) =>
  mount(SetupRestartBanner, {
    props: { status: s },
    global: { plugins: [i18n], stubs: { SetupRestarting: { template: "<div>restarting</div>" } } },
  });

describe("SetupRestartBanner", () => {
  test("counts what is waiting and offers the restart", () => {
    const wrapper = mountBanner(status());
    expect(wrapper.text()).toContain("2 changes apply after a restart");
    expect(wrapper.find("button").text()).toContain("Restart Dozzle");
    expect(wrapper.find("pre").exists()).toBe(false);
  });

  test("one change reads as one", () => {
    expect(mountBanner(status({ pending: { authProvider: "simple" } })).text()).toContain(
      "1 change applies after a restart",
    );
  });

  test("nothing pending, nothing shown", () => {
    expect(mountBanner(status({ pending: {} })).text()).toBe("");
  });

  test("restarts Dozzle the same way the wizard does", async () => {
    const wrapper = mountBanner(status());
    await wrapper.find("button").trigger("click");
    await flushPromises();
    expect(calls).toContain("POST /api/setup/restart");
    expect(wrapper.text()).toContain("restarting");
  });

  test("when Dozzle can't restart itself it hands over the compose lines instead", () => {
    const wrapper = mountBanner(status({ canRestart: false }));
    expect(wrapper.text()).not.toContain("Restart Dozzle");
    const snippet = wrapper.find("pre").text();
    expect(snippet).toContain('DOZZLE_ENABLE_ACTIONS: "true"');
    expect(snippet).toContain('DOZZLE_ENABLE_SHELL: "true"');
  });

  test("when this account may not restart it says why, without the compose lines", () => {
    const wrapper = mountBanner(status({ canWrite: false }));
    expect(wrapper.text()).not.toContain("Restart Dozzle");
    expect(wrapper.text()).not.toContain("Dozzle can't restart itself here.");
    expect(wrapper.find("pre").exists()).toBe(false);
    expect(wrapper.text()).toContain("Your account can't change these settings.");
  });

  test("without a login and outside the window it points at the window", () => {
    const wrapper = mountBanner(status({ canWrite: false, authProvider: "none" }));
    expect(wrapper.text()).toContain("Setup window closed.");
    expect(wrapper.find("pre").exists()).toBe(false);
  });

  test("counts the schedule the restart starts, as the list shows it", () => {
    const wrapper = mountBanner(
      status({
        pending: { enableActions: true },
        autoUpdate: { mode: "daily", time: "03:00", supported: true, image: "amir20/dozzle", currentVersion: "v10" },
      }),
    );
    expect(wrapper.text()).toContain("2 changes apply after a restart");
  });
});
