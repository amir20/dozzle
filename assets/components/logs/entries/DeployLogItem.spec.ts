/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import DeployLogItem from "./DeployLogItem.vue";
import { DeployLogEntry } from "@/models/LogEntry";
import type { ContainerUpdate, DeployVerdict } from "@/models/ContainerUpdate";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", enableActions: true, hosts: [{ name: "localhost", id: "localhost" }] },
  withBase: (path: string) => path,
}));

const update: ContainerUpdate = {
  host: "localhost",
  name: "app",
  oldId: "old",
  newId: "new",
  imageRef: "app:latest",
  fromDigest: "sha256:aaaaaaaaaaaa1111",
  toDigest: "sha256:bbbbbbbbbbbb2222",
  at: "2026-10-03T03:00:00Z",
  source: "schedule",
};

function mountMarker(verdict?: Partial<DeployVerdict>, source = update.source) {
  const entry = new DeployLogEntry({ ...update, source }, new Date(update.at));
  if (verdict) entry.verdict = { deployId: "d1", verdict: "regressed", ...verdict } as DeployVerdict;
  return mount(DeployLogItem, {
    props: { logEntry: entry },
    global: {
      stubs: { LogItem: { template: "<div><slot /></div>" } },
      mocks: { $t: (key: string) => key },
    },
  });
}

const cloudLink = (wrapper: ReturnType<typeof mountMarker>) => wrapper.find("a");

describe("<DeployLogItem />", () => {
  test("shows the images and who made the update", () => {
    const wrapper = mountMarker();
    expect(wrapper.text()).toContain("latest (aaaaaaaaaaaa)");
    expect(wrapper.text()).toContain("latest (bbbbbbbbbbbb)");
    expect(wrapper.text()).toContain("update-marker.by.schedule");
    expect(wrapper.find("button").exists()).toBe(false);
  });

  // Rolling back is a Dozzle Cloud action, so a regression links there instead
  // of offering a button here.
  test("a regression nobody has decided links to Dozzle Cloud to roll back", () => {
    const wrapper = mountMarker({ verdict: "regressed", url: "https://cloud.dozzle.dev/updates/d1" });
    expect(wrapper.text()).toContain("update-marker.verdict.regressed");
    const link = cloudLink(wrapper);
    expect(link.attributes("href")).toBe("https://cloud.dozzle.dev/updates/d1");
    expect(link.text()).toContain("update-marker.roll-back");
    expect(wrapper.find("button").exists()).toBe(false);
  });

  test("once decided, the link only opens the update", () => {
    const wrapper = mountMarker({ verdict: "regressed", decision: "rolled_back", url: "https://cloud.dozzle.dev/u" });
    expect(wrapper.text()).toContain("update-marker.decision.rolled_back");
    expect(cloudLink(wrapper).text()).toContain("label.alert-view-in-cloud");
    expect(cloudLink(wrapper).text()).not.toContain("update-marker.roll-back");
  });

  test("a verdict this build does not know shows no raw key", () => {
    const wrapper = mountMarker({ verdict: "something_new" as DeployVerdict["verdict"], reason: "why" });
    expect(wrapper.text()).not.toContain("update-marker.verdict.");
    expect(wrapper.text()).toContain("why");
  });

  // Only Dozzle Cloud knows an update with no source; it shows none.
  test("an unknown source shows no label", () => {
    const wrapper = mountMarker(undefined, "");
    expect(wrapper.text()).not.toContain("update-marker.by.");
  });
});
