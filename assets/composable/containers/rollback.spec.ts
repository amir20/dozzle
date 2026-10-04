/**
 * @vitest-environment jsdom
 */
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { rollbackLabel } from "@/models/Container";

vi.mock("@/stores/config", () => ({
  default: { enableActions: true, hosts: [] },
  withBase: (path: string) => path,
}));

const { composeManaged, holdsData, isRollingBack, loadRollbackTarget, markRollingBack, rollbackTargetOf } =
  await import("./rollback");

const target = {
  imageId: "sha256:4f2a9c1d0e3b5a6b7c8d9e0f",
  ref: "ghcr.io/immich-app/immich-server@sha256:abcdef1234567890",
};

let seq = 0;
// A container id no other test has asked about, since answers are kept per id.
const fresh = (extra: Record<string, unknown> = {}) => ({
  host: "agent-1",
  id: `c${++seq}`,
  isSwarm: false,
  state: "running" as const,
  ...extra,
});

function respond(status: number, body?: unknown) {
  const fetch = vi.fn().mockResolvedValue({ status, json: async () => body });
  vi.stubGlobal("fetch", fetch);
  return fetch;
}

describe("rollback target", () => {
  beforeEach(() => vi.useFakeTimers({ toFake: ["setTimeout"] }));
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  test("comes from the server, for an agent host like a local one", async () => {
    const fetch = respond(200, target);
    const c = fresh();
    expect(rollbackTargetOf(c)).toBeUndefined();

    await loadRollbackTarget(c);
    expect(fetch).toHaveBeenCalledWith(`/api/hosts/agent-1/containers/${c.id}/rollback-target`);
    expect(rollbackTargetOf(c)).toEqual(target);

    await loadRollbackTarget(c);
    expect(fetch).toHaveBeenCalledTimes(1);
  });

  test("none is asked again later, in case the update was recorded after the first ask", async () => {
    const fetch = respond(204);
    const c = fresh();
    await loadRollbackTarget(c);
    expect(rollbackTargetOf(c)).toBeUndefined();

    await loadRollbackTarget(c);
    expect(fetch).toHaveBeenCalledTimes(1);

    vi.advanceTimersByTime(60_000);
    respond(200, target);
    await loadRollbackTarget(c);
    expect(rollbackTargetOf(c)).toEqual(target);
  });

  test("is not asked for a swarm task or a deleted container", async () => {
    const fetch = respond(200, target);
    await loadRollbackTarget(fresh({ isSwarm: true }));
    await loadRollbackTarget(fresh({ state: "deleted" }));
    expect(fetch).not.toHaveBeenCalled();
  });

  test("a failed ask is asked again", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("offline")));
    const c = fresh();
    await loadRollbackTarget(c);
    expect(rollbackTargetOf(c)).toBeUndefined();

    respond(200, target);
    await loadRollbackTarget(c);
    expect(rollbackTargetOf(c)).toEqual(target);
  });
});

describe("rolling back", () => {
  test("is shared by every surface for the container", () => {
    const c = fresh();
    expect(isRollingBack(c)).toBe(false);
    markRollingBack({ host: c.host, id: c.id }, true);
    expect(isRollingBack(c)).toBe(true);
    markRollingBack(c, false);
    expect(isRollingBack(c)).toBe(false);
  });
});

describe("rollbackLabel", () => {
  test("names the repository and a short digest", () => {
    expect(rollbackLabel(target)).toBe("immich-server@sha256:abcdef123456");
  });

  test("falls back to the short image id", () => {
    expect(rollbackLabel({ imageId: "sha256:4f2a9c1d0e3b5a6b7c8d" })).toBe("4f2a9c1d0e3b");
  });
});

describe("rollback warnings", () => {
  const mount = (type: string, source: string, rw = true) => ({ type, source, destination: "/x", rw });

  test("data lives in writable volumes and bind mounts, not the engine socket", () => {
    expect(holdsData({ mounts: [mount("volume", "pgdata")] } as never)).toBe(true);
    expect(holdsData({ mounts: [mount("bind", "/srv/data")] } as never)).toBe(true);
    expect(holdsData({ mounts: [mount("bind", "/var/run/docker.sock")] } as never)).toBe(false);
    expect(holdsData({ mounts: [mount("volume", "config", false)] } as never)).toBe(false);
    expect(holdsData({ mounts: [] } as never)).toBe(false);
  });

  test("compose drift applies to compose projects", () => {
    expect(composeManaged({ labels: { "com.docker.compose.project": "media" } } as never)).toBe(true);
    expect(composeManaged({ labels: {} } as never)).toBe(false);
  });
});
