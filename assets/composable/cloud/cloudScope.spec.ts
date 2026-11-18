import { describe, expect, test, vi } from "vitest";
import { computed, reactive, ref } from "vue";

const view = ref({ containers: [{ id: "a1", name: "api", host: "h1" }], hosts: ["h1"], lines: [{ message: "x" }] });
vi.mock("@/composable/logs/viewContext", () => ({ useViewContext: () => computed(() => view.value) }));
vi.mock("@/stores/container", () => ({ useContainerStore: () => ({ visibleContainers: new Array(14) }) }));
const route = reactive({ path: "/container/a1" });
vi.mock("vue-router", () => ({ useRoute: () => route }));

import { useCloudScope } from "./cloudScope";

describe("useCloudScope", () => {
  test("starts on the containers on screen and widens to the instance", () => {
    const scope = useCloudScope();
    expect(scope.scoped.value).toBe(true);
    expect(scope.containers.value.map((c) => c.id)).toEqual(["a1"]);
    expect(scope.total.value).toBe(14);

    scope.widen();
    expect(scope.scoped.value).toBe(false);
    expect(scope.containers.value).toEqual([]);
    // A turn asked while widened is about the instance, not the page's container.
    expect(scope.scopedView.value.containers).toEqual([]);
    expect(scope.scopedView.value.lines).toBeUndefined();

    scope.narrow();
    expect(scope.containers.value.map((c) => c.id)).toEqual(["a1"]);
  });

  test("navigating to another page drops the widening", () => {
    const scope = useCloudScope();
    scope.widen();
    route.path = "/container/b2";
    view.value = { containers: [{ id: "b2", name: "worker", host: "h1" }], hosts: ["h1"], lines: [] };
    expect(scope.scoped.value).toBe(true);
    expect(scope.containers.value.map((c) => c.id)).toEqual(["b2"]);
  });

  test("a container starting or stopping on the same page keeps the widening", () => {
    route.path = "/host/h1";
    view.value = { containers: [{ id: "a1", name: "api", host: "h1" }], hosts: ["h1"], lines: [] };
    const scope = useCloudScope();
    scope.widen();
    view.value = {
      containers: [
        { id: "a1", name: "api", host: "h1" },
        { id: "c3", name: "cron", host: "h1" },
      ],
      hosts: ["h1"],
      lines: [],
    };
    expect(scope.widened.value).toBe(true);
    expect(scope.containers.value).toEqual([]);
  });

  test("a page with no containers is the instance and cannot be widened", () => {
    view.value = { containers: [], hosts: [], lines: [] };
    const scope = useCloudScope();
    scope.widen();
    expect(scope.scoped.value).toBe(false);
    expect(scope.widened.value).toBe(false);
  });
});
