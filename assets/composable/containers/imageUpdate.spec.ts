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
    selfContainerId: "5e1f00000000" + "0".repeat(52),
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
vi.mock("@/composable/app/storage", async () => {
  const { ref: vueRef } = await import("vue");
  holder.dismissed = vueRef(new Set<string>());
  return { dismissedImageUpdates: holder.dismissed };
});
vi.mock("@/stores/settings", async () => {
  const { ref: vueRef } = await import("vue");
  holder.showAlertSetting = vueRef(false);
  return { showImageUpdateAlert: holder.showAlertSetting };
});
vi.mock("@/composable/app/toast", () => ({
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

const { canUpdate, useImageUpdate, useImageUpdates } = await import("./imageUpdate");

let counter = 0;

// Short form of config.selfContainerId, the way container ids reach the page.
const SELF_ID = "5e1f00000000";

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
    state: "running",
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

  // Dozzle updates itself through a helper container.
  test("marks a standalone Dozzle container as updatable self", async () => {
    mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
    const { result } = await run(container({ id: SELF_ID, image: "amir20/dozzle:latest" }));

    expect(result.showAlert.value).toBe(true);
    expect(result.updatable.value).toBe(true);
    expect(result.isSelf.value).toBe(true);
  });

  // Matched by id like the backend, so a second Dozzle on the same host is ordinary.
  test("does not treat another Dozzle container on the local host as self", async () => {
    mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
    const { result } = await run(container({ image: "amir20/dozzle:latest" }));

    expect(result.isSelf.value).toBe(false);
  });

  // Without its own container id the backend refuses anything that looks like
  // Dozzle, so no button that cannot work is offered.
  test("does not offer updating a local Dozzle container when its own id is unknown", async () => {
    const self = holder.config.selfContainerId;
    holder.config.selfContainerId = undefined;
    try {
      mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
      const { result } = await run(container({ image: "amir20/dozzle:latest" }));
      expect(result.updatable.value).toBe(false);

      mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
      const other = await run(container());
      expect(other.result.updatable.value).toBe(true);
    } finally {
      holder.config.selfContainerId = self;
    }
  });

  // A Dozzle agent on another host is an ordinary container: updating it does
  // not stop the instance doing the updating.
  test("treats a Dozzle agent on a remote host as updatable", async () => {
    mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
    const { result } = await run(container({ id: SELF_ID, image: "amir20/dozzle:latest", host: "remote" }));

    expect(result.isSelf.value).toBe(false);
    expect(result.updatable.value).toBe(true);
  });

  // The server refuses updates in k8s mode, so the menu must not offer one.
  test("does not offer updating in Kubernetes mode", async () => {
    holder.config.mode = "k8s";
    try {
      mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
      const { result } = await run(container({ host: "remote" }));
      expect(result.updatable.value).toBe(false);
    } finally {
      delete holder.config.mode;
    }
  });

  test("allows updating Dozzle when it runs as a swarm service", async () => {
    mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
    const { result } = await run(container({ id: SELF_ID, image: "amir20/dozzle:latest", isSwarm: true }));

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

    // Dozzle replaces itself through a helper, so the update waits for the new process.
    test("updates a standalone Dozzle container as itself", async () => {
      holder.showAlertSetting!.value = true;
      mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
      await run(container({ id: SELF_ID, image: "amir20/dozzle:latest" }));

      holder.toasts[0].action.handler();
      expect(holder.update).toHaveBeenCalledWith({ self: true });
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

    // In k8s there is no per-pod update to unlock, so the notice stays informational.
    test("neither offers an update nor suggests actions in Kubernetes mode", async () => {
      holder.config.mode = "k8s";
      holder.config.enableActions = false;
      holder.showAlertSetting!.value = true;
      try {
        mockCheck({ status: "update-available", remoteDigest: "sha256:new" });
        await run(container());

        expect(holder.toasts[0].action).toBeUndefined();
        expect(holder.toasts[0].message).not.toContain("alert.image-update.enable-actions");
      } finally {
        delete holder.config.mode;
      }
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

describe("useImageUpdates", () => {
  afterEach(() => vi.unstubAllGlobals());

  async function checkAll(...containers: Container[]) {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () =>
          containers.map((c) => ({
            host: c.host,
            id: c.id,
            result: { status: "update-available", image: c.image, checkedAt: new Date().toISOString() },
          })),
      }),
    );
    const updates = useImageUpdates();
    await updates.checkAll(true);
    return updates;
  }

  test("lists a stopped standalone container, but never offers to update it", async () => {
    const stopped = container({ state: "exited" });
    const { hasUpdate } = await checkAll(stopped);
    expect(hasUpdate(stopped)).toBe(true);
    expect(canUpdate(stopped)).toBe(false);
    expect(canUpdate(container({ state: "running" }))).toBe(true);
  });

  test("skips an exited swarm task but offers the running one", async () => {
    const old = container({ state: "exited", isSwarm: true });
    const current = container({ state: "running", isSwarm: true });
    const { hasUpdate } = await checkAll(old, current);
    expect(hasUpdate(old)).toBe(false);
    expect(hasUpdate(current)).toBe(true);
  });

  test("skips a deleted container", async () => {
    const deleted = container({ state: "deleted" });
    const { hasUpdate } = await checkAll(deleted);
    expect(hasUpdate(deleted)).toBe(false);
  });
});
