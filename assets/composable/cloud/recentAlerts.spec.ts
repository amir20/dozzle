/**
 * @vitest-environment jsdom
 */
import { describe, expect, test, vi, beforeEach, afterEach } from "vitest";
import { ref, computed } from "vue";
import type { CloudAlert } from "./cloudAlerts";

// Reactive so a test can link and unlink the instance the way the settings card
// does: in place, without a reload.
type CloudLink = { linked: boolean };
const holder = vi.hoisted(() => ({ cloudConfig: null as ReturnType<typeof import("vue").ref<CloudLink>> | null }));
vi.mock("./cloudConfig", async () => {
  const { ref: vueRef } = await import("vue");
  holder.cloudConfig = vueRef<CloudLink>({ linked: false });
  return { useCloudConfig: () => ({ cloudConfig: holder.cloudConfig }) };
});

// The watermark is per-reader, not per-instance, so it lives in profile storage.
// A plain ref keeps the test off localStorage and lets it be reset per case.
const watermark = vi.hoisted(() => ({ ref: null as ReturnType<typeof import("vue").ref<number>> | null }));
vi.mock("@/composable/app/profileStorage", async () => {
  const { ref: vueRef } = await import("vue");
  return {
    useProfileStorage: () => (watermark.ref ??= vueRef(0)),
  };
});

const ns = (n: number) => n * 1_000_000;

function alert(overrides: Partial<CloudAlert> = {}): CloudAlert {
  return {
    alertId: "a1",
    containerId: "abc",
    hostId: "h",
    ts: ns(100),
    headline: "Pool exhausted",
    level: "error",
    eventCount: 3,
    createdAt: ns(100),
    isOrigin: true,
    ...overrides,
  };
}

const setLinked = (linked: boolean) => (holder.cloudConfig!.value = { linked });

function respondWith(hits: CloudAlert[]) {
  global.fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ hits }) });
}

/** Fresh module state per test: everything here is shared across the page. */
async function freshModule() {
  vi.resetModules();
  watermark.ref = null;
  const mod = await import("./recentAlerts");
  setLinked(false);
  return mod;
}

describe("useRecentAlerts", () => {
  beforeEach(() => {
    respondWith([]);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  test("asks cloud for nothing while the instance is unlinked", async () => {
    const { useRecentAlerts } = await freshModule();
    const { fetchRecentAlerts, alerts } = useRecentAlerts();

    await fetchRecentAlerts();

    expect(global.fetch).not.toHaveBeenCalled();
    expect(alerts.value).toEqual([]);
  });

  test("loads the history once the instance links, without waiting for a poll", async () => {
    // The boot race: cloudConfig is fetched asynchronously, so the history's own
    // fetch usually runs while the instance still looks unlinked.
    const { useRecentAlerts } = await freshModule();
    const { fetchRecentAlerts, alerts } = useRecentAlerts();

    await fetchRecentAlerts();
    expect(alerts.value).toEqual([]);

    respondWith([alert()]);
    setLinked(true);
    await vi.waitFor(() => expect(alerts.value).toHaveLength(1));
  });

  test("drops the history when the instance is unlinked in place", async () => {
    // Unlinking does not reload the page, so without this the notifications
    // history went on listing alerts under a line saying nothing is remembered.
    const { useRecentAlerts } = await freshModule();
    setLinked(true);
    respondWith([alert()]);

    const { fetchRecentAlerts, alerts, byContainer, unseen } = useRecentAlerts();
    await fetchRecentAlerts();
    expect(alerts.value).toHaveLength(1);

    setLinked(false);
    expect(alerts.value).toEqual([]);
    expect(byContainer.value.size).toBe(0);
    expect(unseen.value).toBe(false);
  });
});

describe("unseenIn", () => {
  beforeEach(() => respondWith([]));
  afterEach(() => vi.restoreAllMocks());

  /** Links, loads the given alerts, and hands back the composable. */
  async function loaded(hits: CloudAlert[]) {
    const { useRecentAlerts } = await freshModule();
    setLinked(true);
    respondWith(hits);
    const api = useRecentAlerts();
    await api.fetchRecentAlerts();
    return api;
  }

  test("is quiet when nothing fired on the containers in view", async () => {
    // The reported bug: the rail's bell lit for an alert on another container,
    // and the panel it opened is scoped to the view, so it showed nothing.
    const { unseenIn, unseen } = await loaded([alert({ containerId: "other" })]);
    const dot = unseenIn(computed(() => new Set(["abc"])));

    expect(dot.value).toBe(false);
    expect(unseen.value).toBe(true);
  });

  test("lights for an alert on a container in view", async () => {
    const { unseenIn } = await loaded([alert({ containerId: "abc" })]);
    expect(unseenIn(computed(() => new Set(["abc"]))).value).toBe(true);
  });

  test("falls back to the instance on a view with no containers of its own", async () => {
    // The home page and settings have nothing to scope to, so the panel shows
    // the instance and the dot has to agree with it.
    const { unseenIn } = await loaded([alert({ containerId: "other" })]);
    expect(unseenIn(computed(() => new Set<string>())).value).toBe(true);
  });

  test("tracks the scope as the reader moves between containers", async () => {
    const { unseenIn } = await loaded([alert({ containerId: "other" })]);
    const ids = ref(new Set(["abc"]));
    const dot = unseenIn(ids);

    expect(dot.value).toBe(false);
    ids.value = new Set(["other"]);
    expect(dot.value).toBe(true);
  });
});

describe("markAlertsSeen", () => {
  beforeEach(() => respondWith([]));
  afterEach(() => vi.restoreAllMocks());

  async function loaded(hits: CloudAlert[]) {
    const { useRecentAlerts } = await freshModule();
    setLinked(true);
    respondWith(hits);
    const api = useRecentAlerts();
    await api.fetchRecentAlerts();
    return api;
  }

  test("reading a scoped panel leaves the instance-wide dot alone", async () => {
    // The rail's panel shows one container. Marking the instance's newest seen
    // from there put out the nav's bell for an alert never on this screen.
    const mine = alert({ alertId: "mine", containerId: "abc", ts: ns(100) });
    const theirs = alert({ alertId: "theirs", containerId: "other", ts: ns(300) });
    const { markAlertsSeen, unseen, unseenIn } = await loaded([mine, theirs]);

    const scope = computed(() => new Set(["abc"]));
    markAlertsSeen(mine);

    expect(unseenIn(scope).value).toBe(false);
    expect(unseen.value).toBe(true);
  });

  test("reading the whole history puts every dot out", async () => {
    const newest = alert({ alertId: "newest", containerId: "other", ts: ns(300) });
    const { markAlertsSeen, unseen } = await loaded([alert({ ts: ns(100) }), newest]);

    markAlertsSeen();
    expect(unseen.value).toBe(false);
  });

  test("never moves the watermark backwards", async () => {
    const older = alert({ alertId: "older", ts: ns(100) });
    const { markAlertsSeen, unseen } = await loaded([older, alert({ alertId: "newer", ts: ns(300) })]);

    markAlertsSeen();
    markAlertsSeen(older);
    expect(unseen.value).toBe(false);
  });
});
