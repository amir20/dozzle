/**
 * @vitest-environment jsdom
 */
import { createPinia, setActivePinia } from "pinia";
import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import { defineComponent } from "vue";

import { usePinnedColumnsInUrl } from "./pinnedColumns";
import { usePinnedLogsStore } from "@/stores/pinned";

// @ts-ignore
import EventSource from "eventsourcemock";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [], enableActions: true },
  withBase: (path: string) => path,
}));

const Blank = defineComponent({ template: "<div/>" });

const Layout = defineComponent({
  setup() {
    usePinnedColumnsInUrl();
  },
  template: "<div/>",
});

async function mountLayout(initial: string) {
  // The pinned store pulls in the container store, which opens its event stream on creation.
  global.EventSource = EventSource;

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: "/container/:id", component: Blank },
      { path: "/stack/:name", component: Blank },
    ],
  });
  await router.replace(initial);
  await router.isReady();

  mount(Layout, { global: { plugins: [router] } });
  await flushPromises();

  return { router, store: usePinnedLogsStore() };
}

describe("pinned columns in the url", () => {
  beforeEach(() => setActivePinia(createPinia()));

  test("seeds the pinned columns from ?columns=", async () => {
    const { store } = await mountLayout("/container/aaa?columns=bbb|ccc");

    expect(store.pinnedContainerIds).toEqual(["bbb", "ccc"]);
  });

  test("writes a pinned container into the query", async () => {
    const { router, store } = await mountLayout("/container/aaa");

    store.pinContainer({ id: "bbb" });
    await flushPromises();
    expect(router.currentRoute.value.fullPath).toBe("/container/aaa?columns=bbb");

    store.pinContainer({ id: "ccc" });
    await flushPromises();
    expect(router.currentRoute.value.query.columns).toBe("bbb|ccc");
  });

  test("drops the query once the last column is closed", async () => {
    const { router, store } = await mountLayout("/container/aaa?columns=bbb&hideMenu=");

    store.unPinContainer({ id: "bbb" });
    await flushPromises();
    expect(router.currentRoute.value.query.columns).toBeUndefined();
    expect(router.currentRoute.value.query.hideMenu).toBe("");
  });

  test("carries the columns onto the next route", async () => {
    const { router, store } = await mountLayout("/container/aaa?columns=bbb");

    await router.push("/stack/api");
    await flushPromises();
    expect(router.currentRoute.value.fullPath).toBe("/stack/api?columns=bbb");
    expect(store.pinnedContainerIds).toEqual(["bbb"]);
  });

  test("a route that carries its own columns wins", async () => {
    const { router, store } = await mountLayout("/container/aaa?columns=bbb");

    await router.push("/container/ddd?columns=eee|fff");
    await flushPromises();
    expect(store.pinnedContainerIds).toEqual(["eee", "fff"]);
    expect(router.currentRoute.value.fullPath).toBe("/container/ddd?columns=eee|fff");
  });

  test("pinning the same container twice keeps one column", async () => {
    const { router, store } = await mountLayout("/container/aaa?columns=bbb");

    store.pinContainer({ id: "bbb" });
    await flushPromises();
    expect(store.pinnedContainerIds).toEqual(["bbb"]);
    expect(router.currentRoute.value.query.columns).toBe("bbb");
  });
});
