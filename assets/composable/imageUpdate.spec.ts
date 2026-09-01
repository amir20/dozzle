/**
 * @vitest-environment jsdom
 */
import { describe, expect, test, vi, beforeEach, afterEach } from "vitest";
import { effectScope, shallowRef } from "vue";
import type { Container } from "@/models/Container";
import type { ImageUpdateResult } from "./imageUpdate";

const holder = vi.hoisted(() => ({
  config: { imageCheckMode: "automatic", enableActions: true } as Record<string, unknown>,
  dismissed: null as ReturnType<typeof import("vue").ref<Set<string>>> | null,
}));

// config is the default export of this module, and withBase lives alongside it.
vi.mock("@/stores/config", () => ({
  default: holder.config,
  withBase: (path: string) => path,
}));
vi.mock("./storage", async () => {
  const { ref: vueRef } = await import("vue");
  holder.dismissed = vueRef(new Set<string>());
  return { dismissedImageUpdates: holder.dismissed };
});

const { useImageUpdate } = await import("./imageUpdate");

function container(overrides: Partial<Container> = {}): Container {
  return {
    id: "abc",
    host: "localhost",
    image: "nginx:latest",
    isSwarm: false,
    ...overrides,
  } as Container;
}

function mockCheck(result: Partial<ImageUpdateResult>) {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ status: "update-available", image: "nginx:latest", checkedAt: "", ...result }),
    }),
  );
}

// Runs the composable inside a scope and waits for the initial check.
async function run(c: Container) {
  const scope = effectScope();
  const result = scope.run(() => useImageUpdate(shallowRef(c)))!;
  await vi.waitFor(() => expect(result.result.value).toBeDefined());
  return { result, scope };
}

describe("useImageUpdate", () => {
  beforeEach(() => {
    holder.config.imageCheckMode = "automatic";
    holder.config.enableActions = true;
    holder.dismissed!.value = new Set();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  test("alerts when the registry has a newer digest", async () => {
    mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
    const { result } = await run(container());

    expect(result.updateAvailable.value).toBe(true);
    expect(result.showAlert.value).toBe(true);
  });

  test("stays quiet when up to date", async () => {
    mockCheck({ status: "up-to-date" });
    const { result } = await run(container());

    expect(result.showAlert.value).toBe(false);
  });

  test("does not check at all when the feature is off", async () => {
    holder.config.imageCheckMode = "off";
    const fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);

    const scope = effectScope();
    scope.run(() => useImageUpdate(shallowRef(container())));

    expect(fetchSpy).not.toHaveBeenCalled();
  });

  test("does not check in the background when set to manual", async () => {
    holder.config.imageCheckMode = "manual";
    const fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);

    const scope = effectScope();
    const result = scope.run(() => useImageUpdate(shallowRef(container())))!;
    expect(fetchSpy).not.toHaveBeenCalled();

    // An explicit request still reaches the server.
    mockCheck({ status: "up-to-date" });
    await result.check(true);
    expect(result.result.value?.status).toBe("up-to-date");
  });

  // The alert is informational, so it survives actions being disabled.
  test("alerts even when actions are disabled", async () => {
    holder.config.enableActions = false;
    mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
    const { result } = await run(container());

    expect(result.showAlert.value).toBe(true);
    expect(result.updatable.value).toBe(false);
  });

  // Dozzle cannot stop itself to update, unless swarm does it.
  test("marks a standalone Dozzle container as not updatable", async () => {
    mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
    const { result } = await run(container({ image: "amir20/dozzle:latest" }));

    expect(result.showAlert.value).toBe(true);
    expect(result.updatable.value).toBe(false);
    expect(result.isSelf.value).toBe(true);
  });

  test("allows updating Dozzle when it runs as a swarm service", async () => {
    mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
    const { result } = await run(container({ image: "amir20/dozzle:latest", isSwarm: true }));

    expect(result.updatable.value).toBe(true);
    expect(result.isSelf.value).toBe(false);
  });

  // Dismissal is keyed on the remote digest so a :latest container goes quiet
  // until the image actually moves again.
  test("dismissal silences only the digest that was dismissed", async () => {
    mockCheck({ status: "update-available", remoteDigest: "sha256:one" });
    const { result } = await run(container());

    result.dismiss();
    expect(result.showAlert.value).toBe(false);

    // A newer digest upstream alerts again.
    result.result.value = {
      status: "update-available",
      image: "nginx:latest",
      remoteDigest: "sha256:two",
      checkedAt: "",
    };
    expect(result.showAlert.value).toBe(true);
  });

  test("a failed check does not surface an alert", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("network down")));

    const scope = effectScope();
    const result = scope.run(() => useImageUpdate(shallowRef(container())))!;
    await vi.waitFor(() => expect(result.checking.value).toBe(false));

    expect(result.showAlert.value).toBe(false);
  });
});
