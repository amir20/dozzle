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
vi.mock("./toast", () => ({
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
});
