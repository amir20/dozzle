/**
 * @vitest-environment jsdom
 */
import { beforeEach, describe, expect, test, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { nextTick } from "vue";
import type { ContainerJson } from "@/types/Container";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [{ name: "localhost", id: "localhost" }], maxLogs: 400 },
  withBase: (path: string) => path,
}));

// The store opens an EventSource the moment it is created, so this has to stand in
// for the real one before the module is imported. Driving the store through actual
// events is the point: the accumulation below was a property of how the handlers
// composed, not of any single function.
class FakeEventSource extends EventTarget {
  static last: FakeEventSource | null = null;
  static readonly OPEN = 1;
  readyState = 1;
  onopen: ((e: Event) => void) | null = null;
  constructor(public url: string) {
    super();
    FakeEventSource.last = this;
  }
  close() {
    this.readyState = 2;
  }
  emit(name: string, payload: unknown) {
    this.dispatchEvent(Object.assign(new Event(name), { data: JSON.stringify(payload) }));
  }
}
vi.stubGlobal("EventSource", FakeEventSource);

const { useContainerStore } = await import("./container");

function json(id: string, host = "localhost", state = "running"): ContainerJson {
  return {
    id,
    created: new Date().toISOString(),
    startedAt: new Date().toISOString(),
    finishedAt: new Date().toISOString(),
    image: "image",
    name: id,
    command: "cmd",
    host,
    labels: {},
    state,
    cpuLimit: 0,
    memoryLimit: 0,
    stats: [],
    mounts: [],
    mountStats: {},
    ports: [],
  } as unknown as ContainerJson;
}

function setup() {
  setActivePinia(createPinia());
  const store = useContainerStore();
  return { store, es: FakeEventSource.last! };
}

describe("container store list reconciliation", () => {
  beforeEach(() => vi.clearAllMocks());

  // Nothing used to remove a container: `destroy` only marks it "deleted", so a tab
  // left open on a host with churn held every container it had ever seen, each with
  // its own stats window. Only a reload freed them.
  test("drops containers a later list for the same host no longer carries", async () => {
    const { store, es } = setup();

    es.emit("containers-changed", [json("a"), json("b"), json("c")]);
    await nextTick();
    expect(store.containers).toHaveLength(3);

    // b and c are destroyed, then the host is re-listed after something starts
    for (const id of ["b", "c"]) {
      es.emit("container-event", { actorId: id, name: "destroy", time: new Date().toISOString() });
    }
    es.emit("containers-changed", [json("a"), json("d")]);
    await nextTick();

    expect(store.containers.map((c) => c.id).sort()).toEqual(["a", "d"]);
  });

  // Only the payload sent on connect covers every host. The rest are one host's list
  // after a start, a rename or a stale-host repair, so treating one as the whole world
  // would evict every other host's containers.
  test("a single host's list never evicts another host", async () => {
    const { store, es } = setup();

    es.emit("containers-changed", [json("a1", "hostA"), json("a2", "hostA"), json("b1", "hostB")]);
    await nextTick();
    expect(store.containers).toHaveLength(3);

    // hostA is re-listed and has lost a2. hostB is not mentioned at all.
    es.emit("containers-changed", [json("a1", "hostA")]);
    await nextTick();

    expect(store.containers.map((c) => c.id).sort()).toEqual(["a1", "b1"]);
  });

  // An empty payload names no host, so there is nothing it can be authoritative about.
  test("an empty list drops nothing", async () => {
    const { store, es } = setup();

    es.emit("containers-changed", [json("a"), json("b")]);
    await nextTick();

    es.emit("containers-changed", []);
    await nextTick();

    expect(store.containers).toHaveLength(2);
  });

  // A container still on the host keeps its identity, and with it the stats history
  // and EMA it has built up. Replacing it on every list would restart both.
  test("a container that survives a list is the same object", async () => {
    const { store, es } = setup();

    es.emit("containers-changed", [json("a")]);
    await nextTick();
    const first = store.containers[0];

    es.emit("containers-changed", [json("a"), json("b")]);
    await nextTick();

    expect(store.containers.find((c) => c.id === "a")).toBe(first);
  });
});
