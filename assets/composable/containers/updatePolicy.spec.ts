import { describe, expect, test, vi } from "vitest";

vi.mock("@/stores/config", () => ({
  default: { base: "", mode: "server" },
  withBase: (path: string) => path,
}));

import { type ContainerUpdatePolicy, heldBack, imageStatusKind, policyLock, riskyContainers } from "./updatePolicy";

const entry = (over: Partial<ContainerUpdatePolicy>): ContainerUpdatePolicy => ({
  host: "nas",
  id: "a",
  name: "a",
  image: "a:latest",
  state: "running",
  policy: "manual",
  source: "default",
  ...over,
});

describe("riskyContainers", () => {
  // Everything would update these without anyone having chosen to.
  test("lists only unchosen containers with named volumes", () => {
    const list = [
      entry({ name: "postgres", volumes: ["pgdata"] }),
      entry({ name: "web" }),
      entry({ name: "chosen", volumes: ["data"], source: "choice" }),
      entry({ name: "labelled", volumes: ["data"], source: "label" }),
      entry({ name: "dozzle", volumes: ["data"], self: true }),
    ];
    expect(riskyContainers(list).map((c) => c.name)).toEqual(["postgres"]);
  });
});

describe("policyLock", () => {
  const open = { persisted: true, canChoose: true };
  test("a label always wins", () => {
    expect(policyLock(entry({ source: "label" }), open)).toBe("label");
  });
  test("Dozzle follows the schedule itself", () => {
    expect(policyLock(entry({ self: true }), open)).toBe("self");
  });
  test("nothing can be saved without a volume", () => {
    expect(policyLock(entry({}), { persisted: false, canChoose: true })).toBe("volume");
  });
  test("or without the right", () => {
    expect(policyLock(entry({}), { persisted: true, canChoose: false })).toBe("access");
    expect(policyLock(undefined, open)).toBe("access");
  });
  test("otherwise it is open", () => {
    expect(policyLock(entry({}), open)).toBeUndefined();
  });
});

test("heldBack is an Automatic that Dozzle only keeps waiting", () => {
  expect(heldBack(entry({ choice: "auto", policy: "manual" }))).toBe(true);
  expect(heldBack(entry({ choice: "auto", policy: "auto" }))).toBe(false);
  expect(heldBack(entry({ choice: "manual" }))).toBe(false);
});

describe("imageStatusKind", () => {
  const result = (status: string) => ({ status, image: "a", checkedAt: "" }) as never;
  test("off is never checked, whatever an old result says", () => {
    expect(imageStatusKind("off", result("update-available"))).toBe("off");
  });
  test("maps every check status", () => {
    expect(imageStatusKind("auto", undefined)).toBe("pending");
    expect(imageStatusKind("auto", result("up-to-date"))).toBe("current");
    expect(imageStatusKind("auto", result("update-available"))).toBe("available");
    expect(imageStatusKind("manual", result("pinned"))).toBe("pinned");
    expect(imageStatusKind("manual", result("not-checkable"))).toBe("local");
    expect(imageStatusKind("manual", result("auth-required"))).toBe("private");
    expect(imageStatusKind("manual", result("skipped"))).toBe("off");
    expect(imageStatusKind("manual", result("unknown"))).toBe("unknown");
  });
});
