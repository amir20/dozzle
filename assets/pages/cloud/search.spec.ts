/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { createPinia } from "pinia";
import { describe, expect, test, vi } from "vitest";
import { createI18n } from "vue-i18n";
import { createMemoryHistory, createRouter } from "vue-router";

vi.mock("@/stores/config", () => ({ __esModule: true, default: { base: "", hosts: [] }, withBase: (p: string) => p }));

const message = vi.hoisted(() => ({ text: "" }));
vi.mock("@/composable/cloud/cloudLogSearch", async () => {
  const { ref } = await import("vue");
  return {
    useCloudLogSearch: () => ({
      results: ref([
        {
          ts: 0,
          hostId: "h",
          containerId: "c",
          containerName: "app",
          message: message.text,
          stream: "stdout",
          level: "info",
        },
      ]),
      loading: ref(false),
      loadingMore: ref(false),
      error: ref(null),
      errorDetail: ref(null),
      available: ref(true),
      hasMore: ref(false),
      tookMs: ref(null),
      volume: ref([]),
      stepNs: ref(0),
      windowStart: ref(0),
      total: ref(null),
      loadMore: () => {},
    }),
  };
});
vi.mock("@/stores/container", async () => {
  const { defineStore } = await import("pinia");
  return { useContainerStore: defineStore("cloud-search-test", () => ({ allContainersById: {} })) };
});
vi.mock("@/composable/cloud/cloudConfig", async () => {
  const { ref } = await import("vue");
  return { useCloudConfig: () => ({ cloudConfig: ref(null) }) };
});

import Search from "./search.vue";

const i18n = createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false, messages: { en: {} } });

async function marks(text: string, q: string) {
  message.text = text;
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/cloud/search", component: Search }],
  });
  await router.push({ path: "/cloud/search", query: { q } });
  const wrapper = mount(Search, {
    global: {
      plugins: [i18n, router, createPinia()],
      stubs: { PageWithLinks: { template: "<div><slot /></div>" }, SearchInline: true, SearchVolume: true },
    },
  });
  return wrapper.findAll("mark").map((m) => m.text());
}

describe("cloud search highlight", () => {
  test("does not match inside the escaped HTML", async () => {
    expect(await marks("don't", "39")).toEqual([]);
  });

  test("matches characters that get escaped", async () => {
    expect(await marks("a < b & c", "< b &")).toEqual(["< b &"]);
  });
});
