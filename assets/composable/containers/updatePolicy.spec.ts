import { describe, expect, test, vi } from "vitest";

vi.mock("@/stores/config", () => ({
  default: { base: "", mode: "server" },
  withBase: (path: string) => path,
}));

import {
  type ContainerUpdatePolicy,
  autoUpdates,
  imageStatusKind,
  labelPolicy,
  resolvePolicy,
  riskyContainers,
  willAutoUpdate,
  useUpdatePolicies,
} from "./updatePolicy";

const entry = (over: Partial<ContainerUpdatePolicy>): ContainerUpdatePolicy => ({
  host: "nas",
  id: "a",
  name: "a",
  image: "a:latest",
  state: "running",
  ...over,
});

// Mirrors updatepolicy.FromLabels and Resolve, which the scheduler follows.
describe("labelPolicy", () => {
  test("reads the label and the older ones", () => {
    expect(labelPolicy({ "dev.dozzle.update": " AUTO " })).toBe("auto");
    expect(labelPolicy({ "dev.dozzle.update": "off" })).toBe("off");
    expect(labelPolicy({ "dev.dozzle.auto-update": "true" })).toBe("auto");
    expect(labelPolicy({ "dev.dozzle.auto-update": "false" })).toBe("manual");
    expect(labelPolicy({ "dev.dozzle.update-check": "false" })).toBe("off");
    expect(labelPolicy({ "dev.dozzle.update": "off", "dev.dozzle.auto-update": "true" })).toBe("off");
    expect(labelPolicy({ "dev.dozzle.update": "sometimes" })).toBeUndefined();
    expect(labelPolicy({})).toBeUndefined();
  });
});

describe("resolvePolicy", () => {
  test("off moves nothing, labelled follows the label, all moves the rest", () => {
    expect(resolvePolicy("auto", "off")).toBe("manual");
    expect(resolvePolicy(undefined, "off")).toBe("manual");
    expect(resolvePolicy("auto", "labelled")).toBe("auto");
    expect(resolvePolicy(undefined, "labelled")).toBe("manual");
    expect(resolvePolicy(undefined, "all")).toBe("auto");
    expect(resolvePolicy("off", "all")).toBe("off");
    expect(resolvePolicy("manual", "all")).toBe("manual");
  });
});

describe("willAutoUpdate", () => {
  test("never updates a stopped container, whatever its label says", () => {
    expect(willAutoUpdate(entry({ state: "exited" }), "all")).toBe(false);
    expect(willAutoUpdate(entry({ state: "exited", label: "auto" }), "all")).toBe(false);
    expect(willAutoUpdate(entry({ state: "exited", label: "auto" }), "labelled")).toBe(false);
    expect(willAutoUpdate(entry({}), "all")).toBe(true);
  });

  test("works from a container's labels too", () => {
    expect(autoUpdates({ labels: { "dev.dozzle.update": "auto" }, state: "running" }, "labelled")).toBe(true);
    expect(autoUpdates({ labels: {}, state: "running" }, "labelled")).toBe(false);
  });
});

describe("riskyContainers", () => {
  // All would update these without anyone having labelled them.
  test("lists only unlabelled running containers with named volumes", () => {
    const list = [
      entry({ name: "postgres", volumes: ["pgdata"] }),
      entry({ name: "web" }),
      entry({ name: "labelled", volumes: ["data"], label: "auto" }),
      entry({ name: "frozen", volumes: ["data"], label: "off" }),
      entry({ name: "stopped", volumes: ["data"], state: "exited" }),
    ];
    expect(riskyContainers(list).map((c) => c.name)).toEqual(["postgres"]);
  });
});

describe("imageStatusKind", () => {
  test("names each check result", () => {
    expect(imageStatusKind(undefined)).toBe("pending");
    expect(imageStatusKind({ status: "up-to-date", image: "a", checkedAt: "" })).toBe("current");
    expect(imageStatusKind({ status: "update-available", image: "a", checkedAt: "" })).toBe("available");
    expect(imageStatusKind({ status: "not-checkable", image: "a", checkedAt: "" })).toBe("local");
    expect(imageStatusKind({ status: "auth-required", image: "a", checkedAt: "" })).toBe("private");
    expect(imageStatusKind({ status: "skipped", image: "a", checkedAt: "" })).toBe("off");
  });
});

describe("useUpdatePolicies", () => {
  test("reads the list", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ mode: "all", containers: [entry({ name: "web" })] }),
    });
    const { policies, fetchPolicies } = useUpdatePolicies();
    await fetchPolicies();
    expect(global.fetch).toHaveBeenCalledWith("/api/updates/policy");
    expect(policies.value?.mode).toBe("all");
    expect(policies.value?.containers.map((c) => c.name)).toEqual(["web"]);
  });
});
