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
    },
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
      if (url === "/api/setup") return new Response(JSON.stringify(status()));
      if (init?.method === "PATCH") return new Response(null, { status: patchStatus });
      return new Response(null, { status: 200 });
    }),
  );
});

afterEach(() => vi.unstubAllGlobals());

const mountForm = (s: SetupStatus, autosave = false) =>
  mount(AutoUpdateForm, { props: { status: s, autosave }, global: { plugins: [i18n] } });

const patches = () => calls.filter((c) => c.init?.method === "PATCH").map((c) => JSON.parse(c.init!.body as string));

describe("AutoUpdateForm", () => {
  test("shows the saved schedule", () => {
    const wrapper = mountForm(status());
    expect((wrapper.find("input[type=checkbox]").element as HTMLInputElement).checked).toBe(true);
    const [when, time] = wrapper.findAll("select");
    expect((when.element as HTMLSelectElement).value).toBe("daily");
    expect((time.element as HTMLSelectElement).value).toBe("03:00");
  });

  test("the wizard saves on Next, not on change", async () => {
    const wrapper = mountForm(status());
    await wrapper.find("select").setValue("weekly");
    const vm = wrapper.vm as unknown as { dirty: boolean; save: () => Promise<boolean> };
    expect(vm.dirty).toBe(true);
    expect(patches()).toEqual([]);

    expect(await vm.save()).toBe(true);
    expect(patches()).toEqual([{ autoUpdate: { mode: "weekly", time: "03:00" } }]);
  });

  test("Settings saves as it changes", async () => {
    const wrapper = mountForm(status(), true);
    await wrapper.find("input[type=checkbox]").setValue(false);
    await flushPromises();
    expect(patches()).toEqual([{ autoUpdate: { mode: "off", time: "03:00" } }]);
  });

  test("a failed save in Settings puts the saved schedule back", async () => {
    patchStatus = 500;
    const wrapper = mountForm(status(), true);
    await wrapper.find("input[type=checkbox]").setValue(false);
    await flushPromises();
    expect((wrapper.find("input[type=checkbox]").element as HTMLInputElement).checked).toBe(true);
    expect(wrapper.text()).toContain("setup.error.generic");
  });

  test("a schedule pinned by DOZZLE_AUTO_UPDATE is read-only and says so", () => {
    const s = status({ locked: { authProvider: false, enableActions: false, enableShell: false, autoUpdate: true } });
    const wrapper = mountForm(s);
    expect(wrapper.find("input[type=checkbox]").attributes("disabled")).toBeDefined();
    expect(wrapper.find("select").attributes("disabled")).toBeDefined();
    expect(wrapper.text()).toContain("setup.actions.locked");
  });

  test("without a volume nothing can be changed", () => {
    const wrapper = mountForm(status({ dataPersisted: false }));
    expect(wrapper.find("input[type=checkbox]").attributes("disabled")).toBeDefined();
    expect(wrapper.text()).toContain("setup.error.no-data");
    expect((wrapper.vm as unknown as { dirty: boolean }).dirty).toBe(false);
  });
});
