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

vi.mock("@/stores/container", () => ({
  useContainerStore: () => ({
    findContainerById: (id: string) => (id === "new" ? { id, host: "localhost", state: "running" } : undefined),
  }),
}));

vi.mock("@/composable/containers/rollback", () => ({
  requestRollback: vi.fn(),
  loadRollbackTarget: vi.fn(),
  isRollingBack: () => false,
  rollbackTargetOf: () => ({ imageId: "sha256:old", ref: "app:1.4.1" }),
}));

const update: ContainerUpdate = {
  host: "localhost",
  name: "app",
  oldId: "old",
  newId: "new",
  fromRef: "app:1.4.1",
  toRef: "app:1.4.2",
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

const rollBack = (wrapper: ReturnType<typeof mountMarker>) =>
  wrapper.findAll("button").filter((b) => b.text().includes("update-marker.roll-back"));

describe("<DeployLogItem />", () => {
  test("offers a rollback on a regression nobody has decided", () => {
    const wrapper = mountMarker({ verdict: "regressed" });
    expect(rollBack(wrapper)).toHaveLength(1);
    expect(wrapper.text()).toContain("update-marker.verdict.regressed");
  });

  // Dozzle Cloud marks the update rolled_back when Dozzle pushes the rollback.
  test("stops offering a rollback once Dozzle Cloud says it was rolled back", () => {
    const wrapper = mountMarker({ verdict: "regressed", decision: "rolled_back" });
    expect(rollBack(wrapper)).toHaveLength(0);
    expect(wrapper.text()).toContain("update-marker.decision.rolled_back");
  });

  test("a verdict this build does not know shows no raw key", () => {
    const wrapper = mountMarker({ verdict: "something_new" as DeployVerdict["verdict"], reason: "why" });
    expect(wrapper.text()).not.toContain("update-marker.verdict.");
    expect(wrapper.text()).toContain("why");
  });

  // The app never names another tool.
  test("an update another updater made reads as made outside Dozzle", () => {
    const wrapper = mountMarker(undefined, "watchtower");
    expect(wrapper.text()).toContain("update-marker.by.external");
    expect(wrapper.text().toLowerCase()).not.toContain("watchtower");
  });
});
