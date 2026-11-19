/**
 * @vitest-environment jsdom
 */
import { enableAutoUnmount, flushPromises, mount } from "@vue/test-utils";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { computed, ref } from "vue";
import { createI18n } from "vue-i18n";
import RailAlerts from "./RailAlerts.vue";
import BarChart from "@/components/ui/BarChart.vue";
import type { AlertBucket, CloudAlert, CloudEvent, FetchedAlerts } from "@/composable/cloud/cloudAlerts";

vi.mock("@/stores/config", () => ({ __esModule: true, default: { enableNotifications: true, hosts: [] } }));

const markAlertsSeen = vi.fn();
const refreshRecentAlerts = vi.fn();
vi.mock("@/composable/cloud/recentAlerts", () => ({
  useRecentAlerts: () => ({ markAlertsSeen, loaded: ref(true), fetchRecentAlerts: vi.fn(), refreshRecentAlerts }),
}));
// What the rail's dot counts in this scope.
const dotted = ref<{ id: string; containerId: string; ts: number }[]>([]);

const fetchAlerts = vi.fn<(...args: unknown[]) => Promise<FetchedAlerts>>();
vi.mock("@/composable/cloud/cloudAlerts", () => ({ useCloudAlerts: () => ({ fetchAlerts }) }));

// What is on screen, and what the panel asks Cloud about. Empty on screen is
// the home page or the chip widened to every container.
const inView = ref<{ host: string; id: string; name: string }[]>([]);
const twoContainers = [
  { host: "h", id: "api", name: "api" },
  { host: "h", id: "worker", name: "worker" },
];
const instance = ref(twoContainers);
vi.mock("@/composable/cloud/cloudScope", () => ({ useCloudScope: () => ({ containers: inView }) }));
vi.mock("@/composable/cloud/cloudFindings", () => ({
  MAX_ASKED: 100,
  useFindingsScope: () => computed(() => (inView.value.length ? inView.value : instance.value)),
}));

vi.mock("@/stores/container", () => ({
  useContainerStore: () => ({ allContainersById: { api: { name: "api" }, worker: { name: "worker" } } }),
}));

const closeRail = vi.fn();
vi.mock("@/composable/cloud/cloudRail", () => ({
  useCloudRail: () => ({ closeRail }),
  useViewAlerts: () => ({ visible: dotted }),
}));
vi.mock("@/composable/cloud/cloudConfig", () => ({
  useCloudConfig: () => ({ cloudStatus: ref({ plan: { retention_days: 30 } }), ensureCloudStatus: vi.fn() }),
}));
vi.mock("@/composable/logs/logJump", () => ({ useLogJump: () => ({ jumpTo: vi.fn() }) }));

const i18n = createI18n({
  legacy: false,
  locale: "en",
  missingWarn: false,
  fallbackWarn: false,
  messages: {
    en: {
      "cloud-rail": {
        "alerts-empty-here": "Nothing has fired on what you're looking at.",
        "alerts-retention":
          "Dozzle Cloud keeps {n} day of alert history. | Dozzle Cloud keeps {n} days of alert history.",
        "create-alert": "Create an alert rule",
        "alerts-matches": "{n} match | {n} matches",
        "alerts-happened": "Happened once | Happened {n} times",
        "alerts-notified": "You were notified once | You were notified {n} times",
        "alerts-notified-earlier": "You were notified earlier",
        "alerts-quiet":
          "{n} more match on {containers}, nothing new to tell you | {n} more matches on {containers}, nothing new to tell you",
        "alerts-quiet-containers": "{n} container | {n} containers",
        "alerts-quiet-alone":
          "{n} match on {containers}, nothing new to tell you | {n} matches on {containers}, nothing new to tell you",
        "alerts-partial": "Not every match is listed here.",
        "alerts-partial-chart": "Not every match is listed here. The chart counts them all.",
        "alerts-capped": "Only the first {n} containers are counted here.",
        "window-24h": "24h",
        "window-7d": "7d",
      },
    },
  },
});

const RouterLinkStub = { props: ["to"], template: "<a data-link><slot /></a>" };
const HOUR = 3600e9;
// Hours ago, in ns. Always today unless the suite runs in the first few hours.
const ago = (h: number) => (Date.now() - h * 3600_000) * 1e6;

const hit = (overrides: Partial<CloudAlert> = {}): CloudAlert => ({
  alertId: "a1",
  containerId: "api",
  hostId: "h",
  ts: ago(2),
  headline: "Pool exhausted",
  level: "error",
  eventCount: 1,
  createdAt: 0,
  isOrigin: true,
  ...overrides,
});
const match = (overrides: Partial<CloudEvent> = {}): CloudEvent => ({
  ts: ago(2),
  containerId: "api",
  message: "pool exhausted",
  alertId: "a1",
  suppressed: true,
  ...overrides,
});

function bucketsFor(hits: CloudAlert[], events: CloudEvent[], count = 24): AlertBucket[] {
  const from = ago(24);
  return Array.from({ length: count }, (_, i) => {
    const start = from + i * HOUR;
    const inside = (ts: number) => ts >= start && ts < start + HOUR;
    return {
      startTs: start,
      matched: events.filter((e) => inside(e.ts)).length,
      notified: hits.filter((h) => h.isOrigin && inside(h.ts)).length,
    };
  });
}

function mountRail() {
  return mount(RailAlerts, {
    global: { plugins: [i18n], stubs: { RouterLink: RouterLinkStub, RelativeTime: true } },
  });
}

enableAutoUnmount(afterEach);
afterEach(() => vi.useRealTimers());

// Rows are grouped by day, so "a few hours ago" must not cross midnight.
beforeEach(() => {
  vi.useFakeTimers({ toFake: ["Date"] });
  vi.setSystemTime(new Date("2026-11-18T15:00:00Z"));
  inView.value = [];
  instance.value = twoContainers;
  fetchAlerts.mockReset();
  markAlertsSeen.mockReset();
  refreshRecentAlerts.mockReset();
  dotted.value = [];
  Element.prototype.scrollIntoView = vi.fn();
});

describe("<RailAlerts />", () => {
  // Nothing in scope still says how far back that reaches and where a rule is made.
  test("an empty scope states the history kept and links to a new rule", async () => {
    inView.value = [{ host: "h", id: "api", name: "api" }];
    fetchAlerts.mockResolvedValue({ alerts: [], events: [] });
    const wrapper = mountRail();
    await flushPromises();

    const empty = wrapper.find("[data-empty]");
    expect(empty.text()).toContain("Nothing has fired on what you're looking at.");
    expect(empty.text()).toContain("Dozzle Cloud keeps 30 days of alert history.");
    expect(wrapper.find("[data-chart]").exists()).toBe(false);

    const create = empty.findComponent(RouterLinkStub);
    expect(create.text()).toBe("Create an alert rule");
    expect(create.props("to")).toEqual({ name: "/notifications", query: { action: "create-alert" } });
    await create.trigger("click");
    expect(closeRail).toHaveBeenCalled();
  });

  // One noisy problem and a scatter of quiet matches read as two rows, and the
  // words are the reader's, not Cloud's.
  test("rows say how often a problem happened and how often the reader was told", async () => {
    const hits = [hit()];
    const events = [
      ...Array.from({ length: 5 }, (_, i) => match({ ts: ago(2) - i * 1e9, logId: i + 1 })),
      match({ alertId: undefined, containerId: "worker", ts: ago(3) }),
      match({ alertId: undefined, containerId: "api", ts: ago(3) }),
    ];
    fetchAlerts.mockResolvedValue({ alerts: hits, events, buckets: bucketsFor(hits, events) });
    const wrapper = mountRail();
    await flushPromises();

    const rows = wrapper.findAll("[data-row]");
    expect(rows).toHaveLength(2);
    expect(rows[0].text()).toContain("Pool exhausted");
    expect(rows[0].find("[data-happened]").text()).toBe("Happened 5 times");
    expect(rows[0].find("[data-notified]").text()).toBe("You were notified once");
    expect(rows[1].find("[data-quiet]").text()).toBe("2 more matches on 2 containers, nothing new to tell you");
    expect(wrapper.text()).not.toMatch(/events|held back|suppressed|absorbed|follow-up/i);

    // Expanding lists the matches, newest first, each with its own jump.
    await rows[0].find("button").trigger("click");
    const lines = rows[0].findAll("[data-occurrences] li");
    expect(lines).toHaveLength(5);
    expect(markAlertsSeen).toHaveBeenCalledWith(hits[0]);
  });

  // Every container on the home page; just the one on a container page, where
  // naming it on every row is noise.
  test("asks about every container with none on screen, or only the one in view", async () => {
    fetchAlerts.mockResolvedValue({ alerts: [hit()], events: [match()] });
    let wrapper = mountRail();
    await flushPromises();
    expect(fetchAlerts.mock.calls[0][0]).toEqual(["api", "worker"]);
    expect(wrapper.find("[data-row]").text()).toContain("api");

    inView.value = [{ host: "h", id: "api", name: "api" }];
    wrapper = mountRail();
    await flushPromises();
    expect(fetchAlerts.mock.calls.at(-1)![0]).toEqual(["api"]);
    expect(wrapper.find("[data-row] .font-mono").exists()).toBe(false);
  });

  // A day at an hour a bar, a week at six; the list and the chart follow.
  test("switching to 7d asks for a week in 28 bars", async () => {
    fetchAlerts.mockResolvedValue({ alerts: [], events: [] });
    const wrapper = mountRail();
    await flushPromises();

    const [, , , first] = fetchAlerts.mock.calls[0] as [string[], Date, Date, Record<string, unknown>];
    expect(first).toMatchObject({ followUps: true, events: true, buckets: 24 });
    const [, from24, to24] = fetchAlerts.mock.calls[0] as [string[], Date, Date];
    expect(to24.getTime() - from24.getTime()).toBe(24 * 3600_000);

    await wrapper.findAll("[role=tab]")[1].trigger("click");
    await flushPromises();
    const [, from, to, opts] = fetchAlerts.mock.calls.at(-1) as [string[], Date, Date, Record<string, unknown>];
    expect(to.getTime() - from.getTime()).toBe(168 * 3600_000);
    expect(opts).toMatchObject({ buckets: 28 });
  });

  // The chart reads out the hovered bar's time, count and problem; a dot marks
  // where a notification went out; a click brings that time's row into view.
  test("chart hover names the problem, dots mark notifications, click scrolls to the row", async () => {
    const hits = [hit()];
    const events = [match(), match({ ts: ago(2) + 1e9 }), match({ alertId: undefined, ts: ago(10) })];
    const buckets = bucketsFor(hits, events);
    fetchAlerts.mockResolvedValue({ alerts: hits, events, buckets });
    const wrapper = mountRail();
    await flushPromises();

    expect(wrapper.find("[data-readout]").text()).toBe("3 matches");
    expect(wrapper.findAll("[data-notified-dot]")).toHaveLength(1);

    const index = buckets.findIndex((b) => b.notified > 0);
    wrapper.findComponent(BarChart).vm.$emit("hoverValue", 2, index, buckets.length);
    await wrapper.vm.$nextTick();
    const readout = wrapper.find("[data-readout]").text();
    expect(readout).toContain("2 matches");
    expect(readout).toContain("Pool exhausted");

    await wrapper.find("[data-chart] button").trigger("click");
    const problem = wrapper.findAll("[data-row]")[0];
    expect(problem.element.scrollIntoView).toHaveBeenCalled();
    expect(problem.classes()).toContain("bg-base-300");

    // The quiet hour lands on the quiet row.
    const quietIndex = buckets.findIndex((b) => b.matched > 0 && b.notified === 0);
    wrapper.findComponent(BarChart).vm.$emit("hoverValue", 1, quietIndex, buckets.length);
    await wrapper.vm.$nextTick();
    expect(wrapper.find("[data-readout]").text()).not.toContain("Pool exhausted");
    await wrapper.find("[data-chart] button").trigger("click");
    expect(wrapper.findAll("[data-row]")[1].classes()).toContain("bg-base-300");
  });

  // An older Cloud counts nothing per bucket; drawing from the capped matches
  // would lie, so there is no chart, only the list.
  test("no chart without buckets", async () => {
    fetchAlerts.mockResolvedValue({ alerts: [hit()], events: [match()] });
    const wrapper = mountRail();
    await flushPromises();
    expect(wrapper.find("[data-chart]").exists()).toBe(false);
    expect(wrapper.findAll("[data-row]")).toHaveLength(1);
  });

  // With nothing above it, a quiet row is not "more" of anything.
  test("a day with only quiet matches drops the 'more'", async () => {
    fetchAlerts.mockResolvedValue({
      alerts: [],
      events: [match({ alertId: undefined }), match({ alertId: undefined })],
    });
    const wrapper = mountRail();
    await flushPromises();
    expect(wrapper.find("[data-quiet]").text()).toBe("2 matches on 1 container, nothing new to tell you");
  });

  // The chart counts every match; the rows stop where Cloud's list did, and
  // the panel says so in the reader's words.
  test("a cut list says the chart counts every match, and nothing when whole", async () => {
    const hits = [hit()];
    const events = [match()];
    fetchAlerts.mockResolvedValue({ alerts: hits, events, buckets: bucketsFor(hits, events), truncated: true });
    let wrapper = mountRail();
    await flushPromises();
    expect(wrapper.find("[data-partial]").text()).toBe("Not every match is listed here. The chart counts them all.");
    expect(wrapper.find("[data-capped]").exists()).toBe(false);

    // No chart from an older Cloud, so no claim about one.
    fetchAlerts.mockResolvedValue({ alerts: hits, events, truncated: true });
    wrapper = mountRail();
    await flushPromises();
    expect(wrapper.find("[data-partial]").text()).toBe("Not every match is listed here.");

    fetchAlerts.mockResolvedValue({ alerts: hits, events, buckets: bucketsFor(hits, events) });
    wrapper = mountRail();
    await flushPromises();
    expect(wrapper.find("[data-partial]").exists()).toBe(false);
  });

  // Past Cloud's cap only the first containers are asked about, and the panel
  // says so whether or not anything matched, since "nothing fired" would read
  // as instance-wide. The flag Cloud sends is then about its list alone.
  test("past the container cap asks for the first 100 and says so", async () => {
    instance.value = Array.from({ length: 101 }, (_, i) => ({ host: "h", id: `c${i}`, name: `c${i}` }));
    fetchAlerts.mockResolvedValue({ alerts: [], events: [] });
    const wrapper = mountRail();
    await flushPromises();

    expect(fetchAlerts.mock.calls[0][0]).toEqual(instance.value.slice(0, 100).map((c) => c.id));
    expect(wrapper.find("[data-empty]").exists()).toBe(true);
    expect(wrapper.find("[data-capped]").text()).toBe("Only the first 100 containers are counted here.");
    expect(wrapper.find("[data-partial]").exists()).toBe(false);
  });

  test("a failed load says nothing about the cap", async () => {
    instance.value = Array.from({ length: 101 }, (_, i) => ({ host: "h", id: `c${i}`, name: `c${i}` }));
    fetchAlerts.mockResolvedValue({ alerts: [], events: [], failed: true });
    const wrapper = mountRail();
    await flushPromises();
    expect(wrapper.find("[data-capped]").exists()).toBe(false);
  });
  // The dot counts recent alerts the timeline can leave out, such as one on a
  // container since recreated. Opening the panel has to clear it anyway.
  test("opening the panel clears what the rail's dot counted, even off the timeline", async () => {
    const gone = { id: "old", containerId: "recreated", ts: ago(1) };
    dotted.value = [gone];
    fetchAlerts.mockResolvedValue({ alerts: [], events: [] });
    mountRail();
    await flushPromises();
    expect(refreshRecentAlerts).toHaveBeenCalled();
    expect(markAlertsSeen).toHaveBeenCalledWith(gone);
  });
  // A follow-up anchor carries a later batch's time; marking it would put out
  // dots for alerts on other containers that were never on screen.
  test("marks only origin alerts seen, never a follow-up anchor", async () => {
    const origin = hit({ ts: ago(3) });
    const anchor = hit({ ts: ago(0.1), isOrigin: false });
    fetchAlerts.mockResolvedValue({ alerts: [origin, anchor], events: [] });
    mountRail();
    await flushPromises();
    expect(markAlertsSeen).toHaveBeenCalledWith(origin);
    expect(markAlertsSeen).not.toHaveBeenCalledWith(anchor);
  });

  test("a panel that failed to load marks nothing seen", async () => {
    dotted.value = [{ id: "a", containerId: "api", ts: ago(1) }];
    fetchAlerts.mockResolvedValue({ alerts: [hit()], events: [], failed: true });
    mountRail();
    await flushPromises();
    expect(markAlertsSeen).not.toHaveBeenCalled();
  });
});
