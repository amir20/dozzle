/**
 * @vitest-environment jsdom
 */
import { describe, expect, test } from "vitest";
import { rollbackLabel, rollbackTarget } from "@/models/Container";
import { composeManaged, holdsData } from "./rollback";

const updated = {
  "dev.dozzle.previous-image": "sha256:4f2a9c1d0e3b5a6b7c8d9e0f",
  "dev.dozzle.previous-ref": "ghcr.io/immich-app/immich-server@sha256:abcdef1234567890",
  "dev.dozzle.update-source": "schedule",
};

describe("rollbackTarget", () => {
  test("is the image the last update replaced", () => {
    expect(rollbackTarget({ labels: updated, isSwarm: false })).toEqual({
      imageId: "sha256:4f2a9c1d0e3b5a6b7c8d9e0f",
      ref: "ghcr.io/immich-app/immich-server@sha256:abcdef1234567890",
    });
  });

  test("is unknown without an update, for a swarm task, and after a rollback", () => {
    expect(rollbackTarget({ labels: {}, isSwarm: false })).toBeUndefined();
    expect(rollbackTarget({ labels: updated, isSwarm: true })).toBeUndefined();
    expect(
      rollbackTarget({ labels: { ...updated, "dev.dozzle.update-source": "rollback" }, isSwarm: false }),
    ).toBeUndefined();
  });

  test("an image built locally has no ref", () => {
    expect(rollbackTarget({ labels: { "dev.dozzle.previous-image": "sha256:aaa" }, isSwarm: false })).toEqual({
      imageId: "sha256:aaa",
      ref: undefined,
    });
  });
});

describe("rollbackLabel", () => {
  test("names the repository and a short digest", () => {
    expect(rollbackLabel(rollbackTarget({ labels: updated, isSwarm: false })!)).toBe(
      "immich-server@sha256:abcdef123456",
    );
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
