/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { afterEach, describe, expect, test, vi } from "vitest";
import { computed, nextTick, ref } from "vue";
import { createI18n } from "vue-i18n";
import RailFindings from "./RailFindings.vue";
import type { CloudFinding } from "@/composable/cloud/cloudFindings";

const finding: CloudFinding = {
  id: "k1",
  severity: "high",
  status: "ongoing",
  locked: false,
  headline: "disk filling up",
  containers: [{ host: "h1", id: "a1", name: "api" }],
  daysOpen: 1,
  firstSeenAt: 1,
  lastSeenAt: 2,
};

const findings = ref<CloudFinding[]>([finding]);
const canIgnore = ref(false);
const truncated = ref(false);
const lastScanAt = ref(0);
const nextScanAt = ref(0);
const scope = ref([{ host: "h1", id: "a1", name: "api" }]);

vi.mock("@/stores/config", () => ({
  __esModule: true,
  // canLinkCloud says nothing about this reader's role; the response does.
  default: { canLinkCloud: true, hosts: [] },
  withBase: (p: string) => p,
}));
vi.mock("@/composable/cloud/cloudScope", () => ({
  useCloudScope: () => ({ containers: computed(() => scope.value) }),
}));
vi.mock("@/composable/cloud/cloudFindings", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/composable/cloud/cloudFindings")>()),
  useFindingsScope: () => computed(() => scope.value),
  useCloudFindings: () => ({
    findings,
    canIgnore,
    truncated,
    lastScanAt,
    nextScanAt,
    loading: ref(false),
    failed: ref(false),
    ignore: vi.fn(),
    unignore: vi.fn(),
    markSeen: vi.fn(),
  }),
}));
vi.mock("@/composable/logs/logJump", () => ({ useLogJump: () => ({ jumpTo: vi.fn() }) }));
vi.mock("@/composable/cloud/cloudRail", () => ({ useCloudRail: () => ({ closeRail: vi.fn() }) }));

const i18n = createI18n({
  legacy: false,
  locale: "en",
  missingWarn: false,
  fallbackWarn: false,
  messages: {
    en: {
      "cloud-findings": {
        ignore: "Ignore",
        truncated: "Not every finding is listed here.",
        instance: "Findings are kept per container.",
        "next-review": "Next review {when}, {time}",
        "last-review": "Last reviewed {when}",
        "first-review": "First review {when}, {time}",
        "first-review-hint": "No findings yet. They show up here after the first review.",
        "empty-new": "Nothing new noticed on what you're looking at.",
        "empty-reviewed": "Nothing noticed on {name} in the last review.",
      },
      "cloud-rail": { "scope-n": "{n} containers" },
    },
  },
});

async function mountExpanded() {
  const wrapper = mount(RailFindings, { global: { plugins: [i18n], stubs: { RelativeTime: true } } });
  await wrapper.find("button[aria-expanded]").trigger("click");
  return wrapper;
}

afterEach(() => {
  findings.value = [finding];
  canIgnore.value = false;
  truncated.value = false;
  lastScanAt.value = 0;
  nextScanAt.value = 0;
  scope.value = [{ host: "h1", id: "a1", name: "api" }];
  vi.useRealTimers();
});

describe("<RailFindings />", () => {
  test("offers Ignore only when the server says this reader may", async () => {
    const wrapper = await mountExpanded();
    expect(wrapper.text()).not.toContain("Ignore");

    canIgnore.value = true;
    await nextTick();
    expect(wrapper.text()).toContain("Ignore");
  });

  test("says when the list stops short", async () => {
    const wrapper = await mountExpanded();
    expect(wrapper.text()).not.toContain("Not every finding");

    truncated.value = true;
    await nextTick();
    expect(wrapper.text()).toContain("Not every finding is listed here.");
  });

  // An empty panel says when the review runs, so "nothing" has an end date.
  test("states the next and last review, in the reader's time", async () => {
    vi.useFakeTimers({ now: new Date("2026-11-18T12:00:00Z"), toFake: ["Date"] });
    nextScanAt.value = new Date("2026-11-18T15:00:00Z").getTime() * 1e6;
    lastScanAt.value = new Date("2026-11-17T15:00:00Z").getTime() * 1e6;
    findings.value = [];
    const wrapper = mount(RailFindings, { global: { plugins: [i18n], stubs: { RelativeTime: true } } });
    const schedule = wrapper.find("[data-schedule]").text();
    expect(schedule).toContain("Next review in 3 hours, 03:00 PM");
    expect(schedule).toContain("Last reviewed yesterday");
    expect(wrapper.text()).toContain("Nothing noticed on api in the last review.");
  });

  // Before any review there is no answer, so the empty line must not read as one.
  test("before the first review the empty line says none has run", () => {
    nextScanAt.value = Date.now() * 1e6 + 3_600_000 * 1e6;
    findings.value = [];
    const wrapper = mount(RailFindings, { global: { plugins: [i18n], stubs: { RelativeTime: true } } });
    expect(wrapper.find("[data-schedule]").text()).toContain("First review");
    expect(wrapper.text()).toContain("No findings yet. They show up here after the first review.");
    expect(wrapper.text()).not.toContain("Nothing new noticed");
    expect(wrapper.find('[role="tablist"]').exists()).toBe(true);
  });

  test("several containers are counted in the empty line", () => {
    lastScanAt.value = Date.now() * 1e6;
    scope.value = [
      { host: "h1", id: "a1", name: "api" },
      { host: "h1", id: "b2", name: "web" },
    ];
    findings.value = [];
    const wrapper = mount(RailFindings, { global: { plugins: [i18n], stubs: { RelativeTime: true } } });
    expect(wrapper.text()).toContain("Nothing noticed on 2 containers in the last review.");
  });

  // With nothing on screen the instance is asked about, so its findings are listed
  // here rather than only in Cloud.
  test("the home page lists the instance's findings, with each one's containers", () => {
    scope.value = [
      { host: "h1", id: "a1", name: "api" },
      { host: "h1", id: "b2", name: "worker" },
    ];
    findings.value = [finding];
    const wrapper = mount(RailFindings, { global: { plugins: [i18n], stubs: { RelativeTime: true } } });
    expect(wrapper.text()).toContain("disk filling up");
    expect(wrapper.text()).toContain("api");
  });

  test("with no schedule from Cloud there is no schedule line", () => {
    findings.value = [];
    const wrapper = mount(RailFindings, { global: { plugins: [i18n], stubs: { RelativeTime: true } } });
    expect(wrapper.find("[data-schedule]").exists()).toBe(false);
  });
});
