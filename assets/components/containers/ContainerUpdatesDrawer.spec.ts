/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { ref } from "vue";
import { createI18n } from "vue-i18n";

vi.mock("@/stores/config", () => ({
  default: { base: "", mode: "server", hosts: [], enableUpdates: true, imageCheckMode: "off" },
  withBase: (path: string) => path,
}));

vi.mock("@/stores/hosts", async () => {
  const { ref } = await import("vue");
  const hosts = ref({
    local: { id: "local", name: "web-1" },
    remote: { id: "remote", name: "web-2" },
  });
  return { useHosts: () => ({ hosts }) };
});

vi.mock("@/stores/container", () => ({ useContainerStore: () => ({}) }));
vi.mock("pinia", async (orig) => ({
  ...(await orig<typeof import("pinia")>()),
  storeToRefs: () => ({ containers: ref([]) }),
}));

vi.mock("@/composable/containers/imageUpdate", () => ({
  useImageUpdates: () => ({
    checkAll: vi.fn(),
    checking: ref(false),
    hasUpdate: () => false,
    isSelf: () => false,
    resultFor: () => undefined,
  }),
}));

vi.mock("@/composable/setup/setup", () => ({
  useSetup: () => ({ status: ref({}), fetchStatus: vi.fn() }),
}));

vi.mock("@/composable/containers/bulkUpdate", async (orig) => {
  const actual = await orig<typeof import("@/composable/containers/bulkUpdate")>();
  const { ref, computed } = await import("vue");
  const job = ref({
    trigger: "manual",
    startedAt: "2026-01-01T00:00:00Z",
    running: true,
    items: [
      { host: "local", id: "a", name: "api", image: "api:latest", status: "done" },
      { host: "remote", id: "b", name: "api", image: "api:latest", status: "error", error: "boom" },
    ],
  });
  return {
    ...actual,
    useBulkUpdate: () => ({ job, running: computed(() => true), start: vi.fn(), hold: () => () => {} }),
  };
});

import ContainerUpdatesDrawer from "./ContainerUpdatesDrawer.vue";

const i18n = createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false, messages: { en: {} } });

describe("<ContainerUpdatesDrawer />", () => {
  test("names each row's host while an update runs", () => {
    const wrapper = mount(ContainerUpdatesDrawer, {
      global: { plugins: [i18n], stubs: { InlineNotice: true, KeyShortcut: true, RouterLink: true } },
    });
    const text = wrapper.text();
    expect(text).toContain("web-1");
    expect(text).toContain("web-2");
  });
});
