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

vi.mock("@/stores/config", () => ({
  __esModule: true,
  // canLinkCloud says nothing about this reader's role; the response does.
  default: { canLinkCloud: true, hosts: [] },
  withBase: (p: string) => p,
}));
vi.mock("@/composable/cloud/cloudScope", () => ({
  useCloudScope: () => ({ containers: computed(() => [{ host: "h1", id: "a1", name: "api" }]) }),
}));
vi.mock("@/composable/cloud/cloudFindings", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/composable/cloud/cloudFindings")>()),
  useCloudFindings: () => ({
    findings,
    canIgnore,
    truncated,
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
  messages: { en: { "cloud-findings": { ignore: "Ignore", truncated: "Not every finding is listed here." } } },
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
});
