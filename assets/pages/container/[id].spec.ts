/** @vitest-environment jsdom */
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Ref } from "vue";
import { createI18n } from "vue-i18n";
import { createMemoryHistory, createRouter, type Router } from "vue-router";
import { Container } from "@/models/Container";
import type { ContainerState } from "@/types/Container";
import { useContainerStore } from "@/stores/container";
import { settings } from "@/stores/settings";
import ContainerPage from "./[id].vue";

vi.mock("@/stores/config", () => ({
  default: { base: "", hosts: [{ id: "host", name: "test host" }] },
  withBase: (path: string) => path,
}));

vi.mock("@/stores/container", async () => {
  const { defineStore } = await import("pinia");
  const { shallowRef, computed, ref } = await import("vue");
  return {
    useContainerStore: defineStore("container-page-test", () => {
      const containers = shallowRef<Container[]>([]);
      const ready = ref(true);
      const currentContainer = (id: Ref<string>) => computed(() => containers.value.find((c) => c.id === id.value));
      return { containers, ready, currentContainer };
    }),
  };
});

const minutesAgo = (m: number) => new Date(Date.now() - m * 60 * 1000);

function container(id: string, state: ContainerState, startedAt: Date, finishedAt: Date) {
  return new Container(id, startedAt, startedAt, finishedAt, "app:1", "app", "/app", "host", {}, state, 0, 0, []);
}

// "old" was recreated a minute ago and "new" replaced it.
const oldRunning = () => container("old", "running", minutesAgo(10), new Date(0));
const oldStopped = () => container("old", "exited", minutesAgo(10), minutesAgo(1));
const newRunning = () => container("new", "running", minutesAgo(1), new Date(0));

let router: Router;

async function open(id: string) {
  router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/container/:id", name: "/container/[id]", component: ContainerPage }],
  });
  await router.push({ name: "/container/[id]", params: { id } });
  mount(
    { template: "<router-view />" },
    {
      global: {
        plugins: [router, createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false })],
        stubs: { Search: true, ContainerLog: true, NotFound: true },
      },
    },
  );
  await flushPromises();
}

const currentId = () => (router.currentRoute.value.params as { id?: string }).id;

describe("container page redirect", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    settings.value.automaticRedirect = "instant";
  });

  test("stays on a container that was already stopped when opened", async () => {
    useContainerStore().containers = [oldStopped(), newRunning()];
    await open("old");
    expect(currentId()).toBe("old");
  });

  test("follows a watched container to its replacement", async () => {
    const store = useContainerStore();
    store.containers = [oldRunning()];
    await open("old");

    store.containers = [oldStopped(), newRunning()];
    await flushPromises();
    expect(currentId()).toBe("new");
  });

  test("switching from the running replacement to the stopped original stays put", async () => {
    useContainerStore().containers = [oldStopped(), newRunning()];
    await open("new");

    await router.push({ name: "/container/[id]", params: { id: "old" } });
    await flushPromises();
    expect(currentId()).toBe("old");
  });
});
