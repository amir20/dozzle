import { describe, expect, test, vi } from "vitest";

vi.mock("@/stores/config", () => ({ __esModule: true, default: {}, withBase: (p: string) => p }));
vi.mock("./cloudConfig", () => ({ useCloudConfig: () => ({ cloudConfig: { value: null } }) }));

import { mergeRecentDeploys, type RecentDeploy } from "./recentDeploys";

const update = (newId: string, at: string, extra: Partial<RecentDeploy> = {}): RecentDeploy => ({
  host: "h1",
  name: "web",
  oldId: "old",
  newId,
  at,
  source: "schedule",
  ...extra,
});

describe("mergeRecentDeploys", () => {
  test("adds what only Cloud remembers and keeps one row for what both have", () => {
    const local = [update("a", "2026-10-21T10:00:00.000000123Z", { fromDigest: "nginx@sha256:11" })];
    const cloud = [
      update("a", "2026-10-21T10:00:00.000000123Z", {
        fromDigest: "sha256:11",
        verdict: { deployId: "x", verdict: "regressed" },
      }),
      update("b", "2026-10-19T10:00:00Z", { verdict: { deployId: "y", verdict: "clean" } }),
    ];

    const merged = mergeRecentDeploys(local, cloud);

    expect(merged.map((u) => u.newId)).toEqual(["a", "b"]);
    // Dozzle's own record wins, with Cloud's verdict on it.
    expect(merged[0].fromDigest).toBe("nginx@sha256:11");
    expect(merged[0].verdict?.verdict).toBe("regressed");
  });

  test("a swap undone onto the same container is its own row", () => {
    const merged = mergeRecentDeploys(
      [update("a", "2026-10-20T10:00:00Z"), update("a", "2026-10-21T10:00:00Z", { oldId: "a", rolledBack: true })],
      [],
    );
    expect(merged).toHaveLength(2);
  });

  test("Cloud's microsecond-rounded time still matches Dozzle's record", () => {
    const merged = mergeRecentDeploys(
      [update("a", "2026-10-21T10:00:00.999999999Z")],
      [update("a", "2026-10-21T10:00:01.000000Z", { verdict: { deployId: "x", verdict: "clean" } })],
    );
    expect(merged).toHaveLength(1);
    expect(merged[0].verdict?.verdict).toBe("clean");
  });

  test("keeps the newest 20", () => {
    const cloud = Array.from({ length: 30 }, (_, i) =>
      update(`c${i}`, new Date(Date.UTC(2026, 9, 1, i)).toISOString()),
    );
    const merged = mergeRecentDeploys([], cloud);
    expect(merged).toHaveLength(20);
    expect(merged[0].newId).toBe("c29");
  });
});
