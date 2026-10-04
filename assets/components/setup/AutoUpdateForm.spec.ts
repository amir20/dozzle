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

import AutoUpdateForm from "./AutoUpdateForm.vue";
import type { SetupStatus } from "@/composable/setup/setup";

const i18n = createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false, messages: { en: {} } });

function status(over: Partial<SetupStatus> = {}): SetupStatus {
  return {
    mode: "server",
    dataPersisted: true,
    authProvider: "simple",
    usersFileExists: true,
    enableActions: true,
    enableShell: false,
    locked: { authProvider: false, enableActions: false, enableShell: false, autoUpdate: false },
    pending: {},
    canRestart: true,
    windowOpen: false,
    canWrite: true,
    autoUpdate: {
      mode: "daily",
      time: "03:00",
      supported: true,
      image: "amir20/dozzle:latest",
      currentVersion: "v11",
      containers: "picked",
    },
    ...over,
  };
}

const policies = {
  mode: "picked",
  persisted: true,
  canChoose: true,
  containers: [
    {
      host: "nas",
      id: "pg",
      name: "postgres",
      image: "postgres:latest",
      state: "running",
      policy: "manual",
      source: "default",
      volumes: ["pgdata"],
    },
    { host: "nas", id: "web", name: "web", image: "nginx", state: "running", policy: "manual", source: "default" },
  ],
};

let calls: { url: string; init?: RequestInit }[];

beforeEach(() => {
  calls = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url: string, init?: RequestInit) => {
      calls.push({ url, init });
      if (url === "/api/updates/policy" && !init?.method) return new Response(JSON.stringify(policies));
      if (url === "/api/setup") return new Response(JSON.stringify(status()));
      return new Response(null, { status: init?.method === "PATCH" ? 204 : 200 });
    }),
  );
});

afterEach(() => vi.unstubAllGlobals());

const mountForm = (s: SetupStatus, autosave = false) =>
  mount(AutoUpdateForm, { props: { status: s, autosave }, global: { plugins: [i18n] } });

describe("AutoUpdateForm", () => {
  test("offers the three ways to choose which containers", () => {
    const radios = mountForm(status()).findAll("input[type=radio]");
    expect(radios.map((r) => (r.element as HTMLInputElement).value)).toEqual(["dozzle", "picked", "all"]);
    expect((radios[1].element as HTMLInputElement).checked).toBe(true);
  });

  // An older server sends no mode at all: that is "picked", today's behaviour.
  test("an unset mode reads as Dozzle and containers I pick", () => {
    const s = status();
    delete s.autoUpdate!.containers;
    const radios = mountForm(s).findAll("input[type=radio]");
    expect((radios[1].element as HTMLInputElement).checked).toBe(true);
  });

  test("Everything lists the containers with named volumes and keeps them manual in one click", async () => {
    const wrapper = mountForm(status());
    await flushPromises();
    expect(wrapper.text()).not.toContain("postgres");

    await wrapper.findAll("input[type=radio]")[2].setValue(true);
    expect(wrapper.text()).toContain("postgres");
    expect(wrapper.text()).not.toContain("web");

    await wrapper.find("button").trigger("click");
    await flushPromises();
    const post = calls.find((c) => c.init?.method === "POST");
    expect(JSON.parse(post!.init!.body as string)).toEqual({
      containers: [{ host: "nas", id: "pg" }],
      policy: "manual",
    });
  });

  test("the wizard saves on Next, with both answers in one request", async () => {
    const wrapper = mountForm(status());
    await wrapper.find("select").setValue("weekly");
    await wrapper.findAll("input[type=radio]")[0].setValue(true);
    const vm = wrapper.vm as unknown as { dirty: boolean; save: () => Promise<boolean> };
    expect(vm.dirty).toBe(true);
    expect(calls.some((c) => c.init?.method === "PATCH")).toBe(false);

    expect(await vm.save()).toBe(true);
    const patch = calls.find((c) => c.init?.method === "PATCH");
    expect(JSON.parse(patch!.init!.body as string)).toEqual({
      autoUpdate: { mode: "weekly", time: "03:00" },
      updateContainers: "dozzle",
    });
  });

  test("Settings saves as it changes", async () => {
    const wrapper = mountForm(status(), true);
    await wrapper.findAll("input[type=radio]")[2].setValue(true);
    await flushPromises();
    const patch = calls.find((c) => c.init?.method === "PATCH");
    expect(JSON.parse(patch!.init!.body as string)).toEqual({ updateContainers: "all" });
  });

  // The schedule is pinned by DOZZLE_AUTO_UPDATE, but which containers has no env var.
  test("an env-locked schedule still lets the containers be chosen", async () => {
    const s = status({ locked: { authProvider: false, enableActions: false, enableShell: false, autoUpdate: true } });
    const wrapper = mountForm(s);
    expect(wrapper.find("select").attributes("disabled")).toBeDefined();
    expect(wrapper.text()).toContain("setup.actions.locked");
    expect(wrapper.find("fieldset").attributes("disabled")).toBeUndefined();
  });

  test("without a volume nothing can be changed", () => {
    const wrapper = mountForm(status({ dataPersisted: false }));
    expect(wrapper.find("select").attributes("disabled")).toBeDefined();
    expect(wrapper.find("fieldset").attributes("disabled")).toBeDefined();
  });
});
