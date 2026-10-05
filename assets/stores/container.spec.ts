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
const { useHosts } = await import("./hosts");

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

  // A start reaches the store only as a re-list, so a container that already exists
  // has to pick up its new run's times from it.
  test("a re-list refreshes when a known container started and finished", async () => {
    const { store, es } = setup();

    es.emit("containers-changed", [json("a", "localhost", "exited")]);
    await nextTick();

    const restarted = { ...json("a"), startedAt: "2030-01-01T00:00:00Z", finishedAt: "2029-12-31T00:00:00Z" };
    es.emit("containers-changed", [restarted]);
    await nextTick();

    expect(store.containers[0].startedAt.toISOString()).toBe("2030-01-01T00:00:00.000Z");
    expect(store.containers[0].finishedAt.toISOString()).toBe("2029-12-31T00:00:00.000Z");
  });

  // The server measures sizes on its own schedule and sends each as an update. A later
  // list without a size must not blank a size already shown.
  test("a measured size arrives as an update and survives a list without one", async () => {
    const { store, es } = setup();

    es.emit("containers-changed", [json("a")]);
    await nextTick();
    expect(store.containers[0].sizeRw).toBeUndefined();

    es.emit("container-updated", { ...json("a"), sizeRw: 2048 });
    await nextTick();
    expect(store.containers[0].sizeRw).toBe(2048);

    es.emit("containers-changed", [json("a"), json("b")]);
    await nextTick();
    expect(store.containers.find((c) => c.id === "a")?.sizeRw).toBe(2048);
  });

  // The Disk column is the layer plus the volumes. Volumes arrive on their own, much
  // later, and an update without them means the container has none any more.
  test("volumes add to the disk total and clear when an update drops them", async () => {
    const { store, es } = setup();

    es.emit("containers-changed", [json("a")]);
    await nextTick();
    expect(store.containers[0].diskTotal).toBeUndefined();

    const volume = { name: "data", destination: "/data", size: 3000, links: 1 };
    es.emit("container-updated", { ...json("a"), volumes: [volume] });
    await nextTick();
    expect(store.containers[0].diskTotal).toBe(3000);

    es.emit("container-updated", { ...json("a"), sizeRw: 100, volumes: [volume] });
    await nextTick();
    expect(store.containers[0].diskTotal).toBe(3100);

    es.emit("container-updated", { ...json("a"), sizeRw: 100 });
    await nextTick();
    expect(store.containers[0].diskTotal).toBe(100);
  });
});

describe("events stream reconnect", () => {
  beforeEach(() => vi.clearAllMocks());

  function reconnect(es: FakeEventSource) {
    es.onopen?.(new Event("open"));
  }

  // Emptying the store on open unmounted every log view gated on a container being
  // there (or on ready), and the remount lost its scroll, older pages and stream.
  test("keeps the containers it has and stays ready until the replay lands", async () => {
    const { store, es } = setup();
    reconnect(es);
    es.emit("containers-changed", [json("a"), json("b")]);
    await nextTick();
    const [a, b] = store.containers;

    reconnect(es);
    await nextTick();
    expect(store.ready).toBe(true);
    expect(store.containers).toEqual([a, b]);

    es.emit("containers-changed", [json("a"), json("b")]);
    await nextTick();
    expect(store.containers[0]).toBe(a);
    expect(store.containers[1]).toBe(b);
  });

  // The list sent on connect covers every host, so a host it no longer names went away
  // (or emptied) while the tab was disconnected and its containers go with it.
  test("the replay drops containers and hosts it no longer carries", async () => {
    const { store, es } = setup();
    reconnect(es);
    es.emit("containers-changed", [json("a1", "hostA"), json("b1", "hostB")]);
    await nextTick();

    reconnect(es);
    es.emit("containers-changed", [json("a1", "hostA")]);
    await nextTick();

    expect(store.containers.map((c) => c.id)).toEqual(["a1"]);
  });

  // The replay is not news. A container that started while the tab was disconnected
  // is not one the person just watched appear.
  test("nothing in the replay is flagged as new, but later lists are", async () => {
    const { store, es } = setup();
    reconnect(es);
    es.emit("containers-changed", [json("a")]);
    await nextTick();

    reconnect(es);
    es.emit("containers-changed", [json("a"), json("b")]);
    await nextTick();
    expect(store.containers.find((c) => c.id === "b")?.isNew).toBe(false);

    es.emit("containers-changed", [json("a"), json("b"), json("c")]);
    await nextTick();
    expect(store.containers.find((c) => c.id === "c")?.isNew).toBe(true);
  });
});

describe("host metrics", () => {
  // The 15s metrics tick used to go out as update-host, which replaces the whole
  // host. The local client's raw Host() has no `available` and no swarm `type`, so
  // every tick marked the local host offline and hid the metrics it carried.
  test("a host-metrics tick leaves availability and type alone", async () => {
    const { es } = setup();
    const { hosts } = useHosts();
    Object.assign(hosts.value.localhost!, { available: true, type: "swarm" });

    es.emit("host-metrics", { id: "localhost", metricsAvailable: true, load1: 0.5, uptime: 3600, diskTotal: 100 });
    await nextTick();

    expect(hosts.value.localhost).toMatchObject({
      available: true,
      type: "swarm",
      metricsAvailable: true,
      load1: 0.5,
      uptime: 3600,
      diskTotal: 100,
    });
  });

  test("a tick for an unknown host adds nothing", async () => {
    const { es } = setup();
    const { hosts } = useHosts();

    es.emit("host-metrics", { id: "ghost", metricsAvailable: true });
    await nextTick();

    expect(hosts.value.ghost).toBeUndefined();
  });
});
