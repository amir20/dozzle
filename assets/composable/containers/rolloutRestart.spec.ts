/**
 * @vitest-environment jsdom
 */
import { describe, expect, test, vi, beforeEach, afterEach } from "vitest";
import type { Container } from "@/models/Container";

const holder = vi.hoisted(() => ({
  config: { mode: "k8s", enableActions: true } as Record<string, unknown>,
  toasts: [] as any[],
}));

vi.mock("@/stores/config", () => ({
  default: holder.config,
  withBase: (path: string) => path,
}));
// The owner-chain parser lives in the k8s store. Only it is used here, so what
// the store module loads (the model, settings, the container store) is stubbed.
vi.mock("@/models/Container", () => ({ Container: class {}, GroupedContainers: class {} }));
vi.mock("@/stores/settings", () => ({ showAllContainers: { value: false } }));
vi.mock("@/stores/container", () => ({ useContainerStore: () => ({}) }));
vi.mock("@/composable/app/toast", () => ({
  useToast: () => ({ showToast: (toast: any) => holder.toasts.push(toast) }),
}));
vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => (params ? `${key}:${JSON.stringify(params)}` : key),
  }),
}));

const { containerWorkload, rolloutWorkload, useRolloutRestart } = await import("./rolloutRestart");

function pod(labels: Record<string, string>) {
  return { labels } as unknown as Container;
}

const deploymentPod = pod({
  "@k8s.namespace": "default",
  "@k8s.owner.count": "2",
  "@k8s.owner.0.kind": "ReplicaSet",
  "@k8s.owner.0.name": "api-6f88b977f4",
  "@k8s.owner.1.kind": "Deployment",
  "@k8s.owner.1.name": "api",
});

beforeEach(() => {
  holder.config.mode = "k8s";
  holder.config.enableActions = true;
  holder.toasts = [];
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("rolloutWorkload", () => {
  test("resolves a pod to the top of its owner chain", () => {
    expect(containerWorkload(deploymentPod)).toEqual({ namespace: "default", kind: "Deployment", name: "api" });
  });

  test("offers only the kinds kubectl can roll out", () => {
    expect(rolloutWorkload("default", "StatefulSet", "db")).toBeDefined();
    expect(rolloutWorkload("default", "DaemonSet", "agent")).toBeDefined();
    expect(rolloutWorkload("default", "CronJob", "nightly")).toBeUndefined();
    expect(rolloutWorkload("default", "ReplicaSet", "api-6f88b977f4")).toBeUndefined();
  });

  // A bare pod has no workload labels at all.
  // An operator's custom resource tops the chain, but the StatefulSet below it is what rolls.
  test("finds the StatefulSet under a custom resource", () => {
    const operatorPod = pod({
      "@k8s.namespace": "monitoring",
      "@k8s.owner.count": "2",
      "@k8s.owner.0.kind": "StatefulSet",
      "@k8s.owner.0.name": "prometheus-main",
      "@k8s.owner.1.kind": "Prometheus",
      "@k8s.owner.1.name": "main",
    });
    expect(containerWorkload(operatorPod)).toEqual({
      namespace: "monitoring",
      kind: "StatefulSet",
      name: "prometheus-main",
    });
  });

  test("returns nothing for a pod without an owner", () => {
    expect(containerWorkload(pod({ "@k8s.namespace": "default" }))).toBeUndefined();
  });

  test("is off outside k8s mode and without actions", () => {
    holder.config.mode = "server";
    expect(containerWorkload(deploymentPod)).toBeUndefined();

    holder.config.mode = "k8s";
    holder.config.enableActions = false;
    expect(containerWorkload(deploymentPod)).toBeUndefined();
  });
});

describe("useRolloutRestart", () => {
  test("posts to the workload route and confirms", async () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(null, { status: 204 }));
    const { rolloutRestart, restarting } = useRolloutRestart();

    await rolloutRestart({ namespace: "default", kind: "Deployment", name: "api" });

    expect(fetchSpy).toHaveBeenCalledWith("/api/k8s/workloads/default/Deployment/api/restart", { method: "POST" });
    expect(holder.toasts).toHaveLength(1);
    expect(holder.toasts[0].type).toBe("info");
    expect(restarting.value).toBe(false);
  });

  // Toasts render their message as HTML.
  test("shows the server's reason, escaped", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response('deployments.apps "api" is forbidden: <b>patch</b>', { status: 500 }),
    );
    const { rolloutRestart } = useRolloutRestart();

    await rolloutRestart({ namespace: "default", kind: "Deployment", name: "api" });

    expect(holder.toasts[0].type).toBe("error");
    expect(holder.toasts[0].message).toContain("&lt;b&gt;patch&lt;/b&gt;");
    expect(holder.toasts[0].message).not.toContain("<b>");
  });
});
