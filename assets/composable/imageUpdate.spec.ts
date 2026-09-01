/**
 * @vitest-environment jsdom
 */
import { describe, expect, test, vi, beforeEach, afterEach } from "vitest";
import { effectScope, shallowRef } from "vue";
import type { Container } from "@/models/Container";
import type { ImageUpdateResult } from "./imageUpdate";

const holder = vi.hoisted(() => ({
  config: {
    imageCheckMode: "automatic",
    enableActions: true,
    hosts: [
      { id: "localhost", type: "local" },
      { id: "remote", type: "agent" },
    ],
  } as Record<string, unknown>,
  dismissed: null as ReturnType<typeof import("vue").ref<Set<string>>> | null,
  showAlertSetting: null as ReturnType<typeof import("vue").ref<boolean>> | null,
  toasts: [] as any[],
  update: vi.fn(),
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
vi.mock("@/stores/settings", async () => {
  const { ref: vueRef } = await import("vue");
  holder.showAlertSetting = vueRef(false);
  return { showImageUpdateAlert: holder.showAlertSetting };
});
vi.mock("./toast", () => ({
  useToast: () => ({
    showToast: (toast: any) => holder.toasts.push(toast),
    removeToast: (id: string) => {
      holder.toasts = holder.toasts.filter((t) => t.id !== id);
    },
  }),
}));
vi.mock("./containerActions", () => ({
  useContainerActions: () => ({ update: holder.update }),
}));
vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => (params ? `${key}:${JSON.stringify(params)}` : key),
  }),
}));

const { useImageUpdate } = await import("./imageUpdate");

let counter = 0;

// Scopes are tracked so each test's watchers are torn down. Without this a
// later test flipping a shared setting would re-trigger earlier instances.
const scopes: ReturnType<typeof effectScope>[] = [];

function newScope() {
  const scope = effectScope();
  scopes.push(scope);
  return scope;
}

// Results are shared per container, so each test uses a fresh id.
function container(overrides: Partial<Container> = {}): Container {
  return {
    id: `container-${counter++}`,
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
      json: async () => ({
        status: "update-available",
        image: "nginx:latest",
        checkedAt: new Date().toISOString(),
        ...result,
      }),
    }),
  );
}

// Runs the composable inside a scope and waits for the initial check.
async function run(c: Container) {
  const scope = newScope();
  const result = scope.run(() => useImageUpdate(shallowRef(c)))!;
  await vi.waitFor(() => expect(result.result.value).toBeDefined());
  return { result, scope };
}

describe("useImageUpdate", () => {
  beforeEach(() => {
    holder.config.imageCheckMode = "automatic";
    holder.config.enableActions = true;
    holder.config.hosts = [
      { id: "localhost", type: "local" },
      { id: "remote", type: "agent" },
    ];
    holder.dismissed!.value = new Set();
    holder.showAlertSetting!.value = false;
    holder.toasts.length = 0;
    holder.update.mockClear();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    scopes.splice(0).forEach((scope) => scope.stop());
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

    const scope = newScope();
    scope.run(() => useImageUpdate(shallowRef(container())));

    expect(fetchSpy).not.toHaveBeenCalled();
  });

  test("does not check in the background when set to manual", async () => {
    holder.config.imageCheckMode = "manual";
    const fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);

    const scope = newScope();
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

  // A Dozzle agent on another host is an ordinary container: updating it does
  // not stop the instance doing the updating.
  test("treats a Dozzle agent on a remote host as updatable", async () => {
    mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
    const { result } = await run(container({ image: "amir20/dozzle:latest", host: "remote" }));

    expect(result.isSelf.value).toBe(false);
    expect(result.updatable.value).toBe(true);
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
    mockCheck({ status: "update-available", remoteDigest: "sha256:two" });
    await result.check(true);
    expect(result.showAlert.value).toBe(true);
  });

  // Historical logs describe a container that is gone.
  test("does not check or notify for historical logs", async () => {
    holder.showAlertSetting!.value = true;
    const fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);

    const scope = newScope();
    scope.run(() => useImageUpdate(shallowRef(container()), true));

    expect(fetchSpy).not.toHaveBeenCalled();
    expect(holder.toasts).toHaveLength(0);
  });

  // An image name is arbitrary text and the notice is rendered as HTML.
  test("escapes the image name in the notification", async () => {
    holder.showAlertSetting!.value = true;
    mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
    await run(container({ image: "<img src=x onerror=alert(1)>:latest" }));

    expect(holder.toasts[0].message).not.toContain("<img");
    expect(holder.toasts[0].message).toContain("&lt;img");
  });

  test("a failed check does not surface an alert", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("network down")));

    const scope = newScope();
    const result = scope.run(() => useImageUpdate(shallowRef(container())))!;
    await vi.waitFor(() => expect(result.checking.value).toBe(false));

    expect(result.showAlert.value).toBe(false);
  });

  // Two components asking about the same container share one request.
  test("shares one request across consumers", async () => {
    const fetchSpy = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ status: "up-to-date", image: "nginx:latest", checkedAt: new Date().toISOString() }),
    });
    vi.stubGlobal("fetch", fetchSpy);

    const c = container();
    const scope = newScope();
    const a = scope.run(() => useImageUpdate(shallowRef(c)))!;
    scope.run(() => useImageUpdate(shallowRef(c)));
    await vi.waitFor(() => expect(a.result.value).toBeDefined());

    expect(fetchSpy).toHaveBeenCalledTimes(1);
  });

  describe("notification", () => {
    test("is not shown unless the setting is on", async () => {
      mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
      await run(container());

      expect(holder.toasts).toHaveLength(0);
    });

    test("is shown once when enabled", async () => {
      holder.showAlertSetting!.value = true;
      mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
      const c = container();
      await run(c);

      expect(holder.toasts).toHaveLength(1);
      expect(holder.toasts[0].message).toContain("nginx:latest");

      // A second consumer of the same container must not re-notify.
      const scope = newScope();
      scope.run(() => useImageUpdate(shallowRef(c)));
      expect(holder.toasts).toHaveLength(1);
    });

    test("offers an update action only when Dozzle can apply it", async () => {
      holder.showAlertSetting!.value = true;
      mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
      await run(container());

      holder.toasts[0].action.handler();
      expect(holder.update).toHaveBeenCalled();
    });

    test("omits the update action for a standalone Dozzle container", async () => {
      holder.showAlertSetting!.value = true;
      mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
      await run(container({ image: "amir20/dozzle:latest" }));

      expect(holder.toasts[0].action).toBeUndefined();
      expect(holder.toasts[0].message).toContain("alert.image-update.self");
    });

    // Actions are off by default, so the notice has to say what to do about it.
    test("explains how to enable actions when they are disabled", async () => {
      holder.config.enableActions = false;
      holder.showAlertSetting!.value = true;
      mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
      await run(container());

      expect(holder.toasts[0].action).toBeUndefined();
      expect(holder.toasts[0].message).toContain("alert.image-update.enable-actions");
    });

    test("does not nag about actions when they are already enabled", async () => {
      holder.showAlertSetting!.value = true;
      mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
      await run(container());

      expect(holder.toasts[0].message).not.toContain("alert.image-update.enable-actions");
    });

    // Dismissing from the notification has to persist, not just close it.
    test("offers a dismiss action that silences the update", async () => {
      holder.showAlertSetting!.value = true;
      mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
      const { result } = await run(container());

      expect(result.showAlert.value).toBe(true);
      holder.toasts[0].secondaryAction!.handler();

      expect(result.showAlert.value).toBe(false);
      expect(holder.dismissed!.value!.has("nginx:latest@sha256:new")).toBe(true);
    });

    // A notice already on screen has to go when the update is dismissed from
    // the toolbar menu, or it lingers with no way to act on it.
    test("dismissing removes a notification that is already showing", async () => {
      holder.showAlertSetting!.value = true;
      mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
      const { result } = await run(container());

      expect(holder.toasts).toHaveLength(1);
      result.dismiss();
      expect(holder.toasts).toHaveLength(0);
    });

    test("is not shown for a dismissed update", async () => {
      holder.showAlertSetting!.value = true;
      holder.dismissed!.value = new Set(["nginx:latest@sha256:new"]);
      mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
      await run(container());

      expect(holder.toasts).toHaveLength(0);
    });
  });
});
