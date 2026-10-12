/**
 * @vitest-environment jsdom
 */
import { createPinia, setActivePinia } from "pinia";
import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import { defineComponent } from "vue";
import { createI18n } from "vue-i18n";

import { usePinnedColumnsInUrl, useVisiblePinnedLogs } from "./pinnedColumns";
import { usePinnedLogsStore } from "@/stores/pinned";
import { useContainerStore } from "@/stores/container";
import type { Container } from "@/models/Container";

import EventSource from "@/testing/eventSourceMock";

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

  test("a link repeating the same id opens one column", async () => {
    const { router, store } = await mountLayout("/container/aaa?columns=bbb|bbb|ccc");

    expect(store.pinnedContainerIds).toEqual(["bbb", "ccc"]);
    await flushPromises();
    expect(router.currentRoute.value.query.columns).toBe("bbb|ccc");
  });

  test("pinning the same container twice keeps one column", async () => {
    const { router, store } = await mountLayout("/container/aaa?columns=bbb");

    store.pinContainer({ id: "bbb" });
    await flushPromises();
    expect(store.pinnedContainerIds).toEqual(["bbb"]);
    expect(router.currentRoute.value.query.columns).toBe("bbb");
  });
});

describe("visible pinned columns", () => {
  beforeEach(() => setActivePinia(createPinia()));

  async function mountColumns(initial: string, pinned: string[]) {
    global.EventSource = EventSource;

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: "/container/:id", name: "/container/[id]", component: Blank },
        { path: "/merged/:ids", name: "/merged/[ids]", component: Blank },
        { path: "/stack/:name", name: "/stack/[name]", component: Blank },
      ],
    });
    await router.replace(initial);
    await router.isReady();

    useContainerStore().containers = ["aaa", "bbb", "ccc"].map((id) => ({ id, host: "local" }) as Container);
    const store = usePinnedLogsStore();
    pinned.forEach((id) => store.pinContainer({ id }));

    let visible!: ReturnType<typeof useVisiblePinnedLogs>;
    const Columns = defineComponent({
      setup() {
        visible = useVisiblePinnedLogs();
      },
      template: "<div/>",
    });
    mount(Columns, { global: { plugins: [router, createI18n({})] } });

    return { router, store, ids: () => visible.value.map(({ id }) => id) };
  }

  test("shows every pinned column when the main view streams none of them", async () => {
    const { ids } = await mountColumns("/stack/api", ["bbb", "ccc"]);

    expect(ids()).toEqual(["bbb", "ccc"]);
  });

  test("hides a pinned column whose container is the container page", async () => {
    const { ids } = await mountColumns("/container/bbb", ["bbb", "ccc"]);

    expect(ids()).toEqual(["ccc"]);
  });

  test("hides a pinned column while it is merged into the stream, and keeps it pinned", async () => {
    const { router, store, ids } = await mountColumns("/container/aaa", ["bbb", "ccc"]);
    expect(ids()).toEqual(["bbb", "ccc"]);

    await router.push("/merged/aaa,bbb");
    expect(ids()).toEqual(["ccc"]);
    expect(store.pinnedContainerIds).toEqual(["bbb", "ccc"]);

    await router.push("/container/aaa");
    expect(ids()).toEqual(["bbb", "ccc"]);
  });
});
