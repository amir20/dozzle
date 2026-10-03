/**
 * @vitest-environment jsdom
 */
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { nextTick, ref, shallowRef } from "vue";
import { Container } from "@/models/Container";
import { useStickyEntity } from "./stickyEntity";

vi.mock("@/stores/container", async () => {
  const { defineStore } = await import("pinia");
  const { ref, computed } = await import("vue");
  return {
    useContainerStore: defineStore("sticky-entity-test-containers", () => {
      const containers = ref<Container[]>([]);
      const ready = ref(true);
      const allContainersById = computed(() => Object.fromEntries(containers.value.map((c) => [c.id, c])));
      return { containers, ready, allContainersById };
    }),
  };
});

function container(id: string) {
  const now = new Date();
  return new Container(id, now, now, now, "img", id, "cmd", "host", {}, "running", 0, 0, []);
}

describe("useStickyEntity", () => {
  beforeEach(() => setActivePinia(createPinia()));

  test("keeps the last value while the key stays the same", async () => {
    const list = ref([{ name: "web" }]);
    const name = ref("web");
    const entity = useStickyEntity(
      () => list.value.find((e) => e.name === name.value),
      () => name.value,
    );
    expect(entity.value?.name).toBe("web");

    list.value = [];
    await nextTick();
    expect(entity.value?.name).toBe("web");
  });

  test("lets go when the key changes", async () => {
    const list = ref([{ name: "web" }]);
    const name = ref("web");
    const entity = useStickyEntity(
      () => list.value.find((e) => e.name === name.value),
      () => name.value,
    );

    list.value = [];
    name.value = "db";
    await nextTick();
    expect(entity.value).toBeUndefined();
  });

  test("is undefined for a key that never resolved", () => {
    const entity = useStickyEntity<{ name: string }>(
      () => undefined,
      () => "web",
    );
    expect(entity.value).toBeUndefined();
  });

  // A container removed while the tab slept is dropped with no destroy event,
  // so nothing else ever tells the held object it is gone.
  test("marks a held member deleted once the store drops it", async () => {
    const store = useContainerStore();
    const [a, b] = [container("a"), container("b")];
    store.containers = [a, b];
    const group = shallowRef<{ name: string; containers: Container[] } | undefined>({
      name: "web",
      containers: [store.allContainersById.a, store.allContainersById.b],
    });
    const entity = useStickyEntity(
      () => group.value,
      () => "web",
    );

    group.value = undefined;
    store.containers = [store.allContainersById.a];
    await nextTick();

    const [heldA, heldB] = entity.value!.containers;
    expect(heldA.state).toBe("running");
    expect(heldB.state).toBe("deleted");
  });

  test("does not mark anything while the store is replaying", async () => {
    const store = useContainerStore();
    store.containers = [container("a")];
    const held = store.allContainersById.a;
    useStickyEntity(
      () => ({ name: "web", containers: [held] }),
      () => "web",
    );

    store.ready = false;
    store.containers = [];
    await nextTick();
    expect(held.state).toBe("running");
  });
});
