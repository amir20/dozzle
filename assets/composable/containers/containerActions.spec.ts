/**
 * @vitest-environment jsdom
 */
import { describe, expect, test, vi, beforeEach } from "vitest";
import { effectScope, shallowRef } from "vue";
import type { Container } from "@/models/Container";

const holder = vi.hoisted(() => ({
  toasts: [] as any[],
}));

vi.mock("@/stores/config", () => ({
  default: { enableActions: true },
  withBase: (path: string) => path,
}));
vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }));
vi.mock("@/composable/app/toast", () => ({
  useToast: () => ({
    showToast: (toast: any) => holder.toasts.push(toast),
    updateToast: (id: string, patch: any) => {
      const existing = holder.toasts.find((t) => t.id === id);
      if (existing) Object.assign(existing, patch);
    },
    removeToast: (id: string) => {
      holder.toasts = holder.toasts.filter((t) => t.id !== id);
    },
  }),
}));

const { useContainerActions } = await import("./containerActions");

// Builds an SSE body the update endpoint would produce.
function sseStream(events: Record<string, unknown>[]) {
  const body = events.map((e) => `event: update-progress\ndata: ${JSON.stringify(e)}\n\n`).join("");
  const bytes = new TextEncoder().encode(body);

  return {
    ok: true,
    body: {
      getReader() {
        let sent = false;
        return {
          read: async () => (sent ? { done: true, value: undefined } : ((sent = true), { done: false, value: bytes })),
          cancel: () => {},
        };
      },
    },
  };
}

function run(events: Record<string, unknown>[]) {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(sseStream(events)));

  const scope = effectScope();
  const actions = scope.run(() =>
    useContainerActions(shallowRef({ id: "abc", host: "localhost", image: "nginx:latest" } as Container)),
  )!;
  return actions;
}

const progressToast = () => holder.toasts.find((t) => t.id === "container-update");

describe("useContainerActions update progress", () => {
  beforeEach(() => {
    holder.toasts = [];
  });

  test("reports a percentage while layers download", async () => {
    const actions = run([
      { status: "pulling", layer: "a", current: 50, total: 100 },
      { status: "pulling", layer: "b", current: 50, total: 100 },
    ]);

    await actions.update();

    // Two layers, half of each, is half overall.
    expect(progressToast()?.progress).toBeCloseTo(50);
  });

  test("keeps counting layers that appear partway through", async () => {
    const actions = run([
      { status: "pulling", layer: "a", current: 100, total: 100 },
      { status: "pulling", layer: "b", current: 0, total: 100 },
    ]);

    await actions.update();
    expect(progressToast()?.progress).toBeCloseTo(50);
  });

  // Layers that are already present report no size and must not distort it.
  test("ignores layers with no reported size", async () => {
    const actions = run([
      { status: "pulling", layer: "cached", current: 0, total: 0 },
      { status: "pulling", layer: "a", current: 25, total: 100 },
    ]);

    await actions.update();
    expect(progressToast()?.progress).toBeCloseTo(25);
  });

  test("shows no bar when nothing needs downloading", async () => {
    const actions = run([{ status: "pulling", layer: "cached", current: 0, total: 0 }]);

    await actions.update();
    expect(progressToast()?.progress).toBeUndefined();
  });

  // Recreating cannot report progress, so the bar should not linger.
  test("drops the bar once the pull finishes", async () => {
    const actions = run([{ status: "pulling", layer: "a", current: 100, total: 100 }, { status: "recreating" }]);

    await actions.update();

    expect(progressToast()?.progress).toBeUndefined();
    expect(progressToast()?.message).toBe("toolbar.update-recreating");
  });

  // The toast renders its message as HTML, and a pull error quotes whatever the
  // registry answered.
  test("escapes the error text before it reaches the toast", async () => {
    const actions = run([{ status: "error", error: 'pull failed: <img src=x onerror="alert(1)">' }]);

    await actions.update();

    const message = holder.toasts.find((t) => t.type === "error")?.message;
    expect(message).toBe("pull failed: &lt;img src=x onerror=&quot;alert(1)&quot;&gt;");
  });

  test("says it is verifying once the new container started", async () => {
    const actions = run([{ status: "recreating" }, { status: "verifying" }]);

    await actions.update();
    expect(progressToast()?.message).toBe("toolbar.update-verifying");
  });

  // The old container is back, so this is not a success, and the reason quotes
  // the engine's own text.
  test("reports a rollback as a warning with the escaped reason", async () => {
    const actions = run([
      { status: "recreating" },
      { status: "verifying" },
      { status: "rolled-back", error: "replacement is <b>unhealthy</b>" },
    ]);

    await actions.update();

    expect(progressToast()).toBeUndefined();
    const toast = holder.toasts.find((t) => t.type === "warning");
    expect(toast?.title).toBe("error.update-failed");
    expect(toast?.message).toBe("toolbar.update-rolled-back<br>replacement is &lt;b&gt;unhealthy&lt;/b&gt;");
  });
});

describe("useContainerActions rollback", () => {
  beforeEach(() => {
    holder.toasts = [];
  });

  test("asks for the target the UI offered, and reports success", async () => {
    const actions = run([{ status: "recreating" }, { status: "verifying" }, { status: "done" }]);

    const done = await actions.rollback("sha256:prev");

    expect(done).toBe(true);
    expect(fetch).toHaveBeenCalledWith("/api/hosts/localhost/containers/abc/actions/rollback?to=sha256%3Aprev", {
      method: "POST",
    });
    expect(holder.toasts.find((t) => t.id === "container-rollback")).toBeUndefined();
    expect(holder.toasts.at(-1)).toMatchObject({ type: "info", message: "rollback.done" });
    expect(actions.actionStates.rollback).toBe(false);
  });

  test("an undone rollback is a warning with the reason", async () => {
    const actions = run([{ status: "recreating" }, { status: "rolled-back", error: "replacement is unhealthy" }]);

    expect(await actions.rollback("sha256:prev")).toBe(false);
    const toast = holder.toasts.find((t) => t.type === "warning");
    expect(toast?.title).toBe("rollback.failed");
    expect(toast?.message).toBe("rollback.undone<br>replacement is unhealthy");
  });

  test("a refusal is an error", async () => {
    const actions = run([{ status: "error", error: "no previous image to roll back to" }]);

    expect(await actions.rollback("sha256:prev")).toBe(false);
    expect(holder.toasts.find((t) => t.type === "error")?.message).toBe("no previous image to roll back to");
  });
});

describe("useContainerActions update watch", () => {
  beforeEach(() => {
    holder.toasts = [];
  });

  test("sends the watch choice as a JSON body", async () => {
    const actions = run([{ status: "done" }]);
    await actions.update({ watchInCloud: true });

    const [, init] = (fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(init.headers).toEqual({ "Content-Type": "application/json" });
    expect(JSON.parse(init.body)).toEqual({ watchInCloud: true });
  });

  test("sends no body when nothing was asked", async () => {
    const actions = run([{ status: "done" }]);
    await actions.update();

    const [, init] = (fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(init).toEqual({ method: "POST" });
  });
});
