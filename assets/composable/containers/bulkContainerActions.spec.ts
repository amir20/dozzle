/**
 * @vitest-environment jsdom
 */
import { describe, expect, test, vi, beforeEach } from "vitest";
import { effectScope } from "vue";
import type { Container } from "@/models/Container";

const holder = vi.hoisted(() => ({
  toasts: [] as any[],
  config: {
    enableActions: true,
    mode: "server",
    selfContainerId: "selfself1234abcd",
    hosts: [{ id: "local", type: "local" }],
  } as Record<string, any>,
}));

vi.mock("@/stores/config", () => ({
  default: holder.config,
  withBase: (path: string) => path,
}));
vi.mock("vue-i18n", () => ({
  useI18n: () => ({ t: (key: string, args?: any) => (args ? `${key} ${JSON.stringify(args)}` : key) }),
}));
vi.mock("@/composable/app/toast", () => ({
  useToast: () => ({ showToast: (toast: any) => holder.toasts.push(toast) }),
}));

const { bulkPlan, runBounded, useBulkContainerActions, BULK_CONCURRENCY } = await import("./bulkContainerActions");

function container(name: string, state: string, id = `${name}000000000000`): Container {
  return { id, name, state, host: "local", isSwarm: false } as unknown as Container;
}

const web = container("web", "running");
const db = container("db", "running");
const old = container("old", "exited");
const fresh = container("fresh", "created");
const paused = container("paused", "paused");
const gone = container("gone", "deleted");
const self = container("dozzle", "running", "selfself1234");

const names = (cs: Container[]) => cs.map((c) => c.name);

beforeEach(() => {
  holder.toasts = [];
  holder.config.mode = "server";
});

describe("bulkPlan", () => {
  const all = [web, db, old, fresh, paused, gone, self];

  test("restart and stop take running containers and skip Dozzle itself", () => {
    for (const action of ["restart", "stop"] as const) {
      const plan = bulkPlan(all, action);
      expect(names(plan.targets)).toEqual(["web", "db"]);
      expect(plan.skippedSelf).toBe(true);
    }
  });

  test("start takes stopped containers, never paused or deleted ones", () => {
    const plan = bulkPlan(all, "start");
    expect(names(plan.targets)).toEqual(["old", "fresh"]);
    expect(plan.skippedSelf).toBe(false);
  });

  test("skippedSelf is false when Dozzle is not in the view", () => {
    expect(bulkPlan([web], "stop").skippedSelf).toBe(false);
  });

  test("a Dozzle on another host is an ordinary container", () => {
    const remote = { ...self, host: "remote" } as Container;
    expect(names(bulkPlan([remote], "restart").targets)).toEqual(["dozzle"]);
  });

  test("a swarm task of Dozzle itself is still skipped", () => {
    const task = { ...self, host: "node-2", isSwarm: true } as Container;
    const plan = bulkPlan([web, task], "stop");
    expect(names(plan.targets)).toEqual(["web"]);
    expect(plan.skippedSelf).toBe(true);
  });

  test("k8s restarts each pod once, however many containers it has", () => {
    holder.config.mode = "k8s";
    try {
      const app = container("app", "running", "ns:pod-a:app");
      const sidecar = container("sidecar", "running", "ns:pod-a:sidecar");
      const other = container("other", "running", "ns:pod-b:app");
      expect(names(bulkPlan([app, sidecar, other], "restart").targets)).toEqual(["app", "other"]);
    } finally {
      holder.config.mode = "server";
    }
  });
});

describe("runBounded", () => {
  test("never has more than the limit in flight", async () => {
    let inFlight = 0;
    let peak = 0;
    const items = Array.from({ length: 11 }, (_, i) => i);
    const failed = await runBounded(items, 4, async () => {
      inFlight++;
      peak = Math.max(peak, inFlight);
      await new Promise((r) => setTimeout(r, 1));
      inFlight--;
      return true;
    });
    expect(peak).toBe(4);
    expect(failed).toEqual([]);
  });

  test("a failure or a throw does not stop the rest", async () => {
    const seen: number[] = [];
    const failed = await runBounded([1, 2, 3, 4, 5], 2, async (n) => {
      seen.push(n);
      if (n === 2) throw new Error("boom");
      return n !== 4;
    });
    expect(seen.sort()).toEqual([1, 2, 3, 4, 5]);
    expect(failed.sort()).toEqual([2, 4]);
  });
});

describe("useBulkContainerActions", () => {
  function setup(containers: Container[]) {
    const scope = effectScope();
    return scope.run(() => useBulkContainerActions(() => containers))!;
  }

  test("offers only actions that would touch something", () => {
    expect(setup([web, paused]).available.value.map((p) => p.action)).toEqual(["restart", "stop"]);
    expect(setup([old]).available.value.map((p) => p.action)).toEqual(["start"]);
    expect(setup([self]).available.value).toEqual([]);
  });

  test("k8s only restarts", () => {
    holder.config.mode = "k8s";
    expect(setup([web, old]).available.value.map((p) => p.action)).toEqual(["restart"]);
  });

  test("posts once per container, at most BULK_CONCURRENCY at a time", async () => {
    let inFlight = 0;
    let peak = 0;
    const urls: string[] = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(async (url: string) => {
        urls.push(url);
        inFlight++;
        peak = Math.max(peak, inFlight);
        await new Promise((r) => setTimeout(r, 1));
        inFlight--;
        return { ok: true };
      }),
    );
    const many = Array.from({ length: 9 }, (_, i) => container(`c${i}`, "running"));
    const { run, running, available } = setup(many);
    const done = run(available.value[0]);
    expect(running.value).toBe("restart");
    await done;

    expect(running.value).toBeUndefined();
    expect(urls).toHaveLength(9);
    expect(urls[0]).toBe("/api/hosts/local/containers/c0000000000000/actions/restart");
    expect(peak).toBe(BULK_CONCURRENCY);
    expect(holder.toasts).toEqual([]);
  });

  test("one error toast names every container that failed", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (url: string) => {
        if (url.includes("/web")) return { ok: false };
        if (url.includes("/db")) throw new Error("network");
        return { ok: true };
      }),
    );
    const ok = container("ok", "running");
    const { run, available } = setup([web, db, ok]);
    await run(available.value.find((p) => p.action === "stop")!);

    expect(holder.toasts).toHaveLength(1);
    expect(holder.toasts[0].type).toBe("error");
    expect(holder.toasts[0].message).toContain("web");
    expect(holder.toasts[0].message).toContain("db");
    expect(holder.toasts[0].message).not.toContain("ok");
  });
});
