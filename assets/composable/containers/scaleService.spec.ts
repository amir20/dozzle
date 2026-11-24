/**
 * @vitest-environment jsdom
 */
import { describe, expect, test, vi, beforeEach } from "vitest";
import type { Container } from "@/models/Container";
import type { Service } from "@/models/Stack";

const holder = vi.hoisted(() => ({ config: { mode: "swarm", enableActions: true } as Record<string, unknown> }));

vi.mock("@/stores/config", () => ({ default: holder.config, withBase: (path: string) => path }));

const { scalableService } = await import("./scaleService");

function service(name: string, ...tasks: string[]) {
  const containers = tasks.map((task) => ({ labels: { "com.docker.swarm.task.name": task } }) as unknown as Container);
  return { name, containers } as unknown as Service;
}

describe("scalableService", () => {
  beforeEach(() => {
    holder.config.mode = "swarm";
    holder.config.enableActions = true;
  });

  test("a replicated service, whose tasks carry a slot number, can be scaled", () => {
    const web = service("shop_web", "shop_web.1.5npjxh2xppihjuju90owxqlas", "shop_web.2.b20z5m2m6jgm027fal2px6hgj");
    expect(scalableService(web)).toEqual({ service: web, global: false });
  });

  // Still offered, so the dialog can say why there is nothing to set.
  test("a global service, whose tasks carry a node id, is marked global", () => {
    const metrics = service("shop_metrics", "shop_metrics.f0fb0ko1xg6a4lcc3mqbe28oo.ks404plkxch4phq1t87itew0u");
    expect(scalableService(metrics)).toEqual({ service: metrics, global: true });
  });

  test("not without actions, in k8s, or without a swarm task", () => {
    const web = service("shop_web", "shop_web.1.abc");
    holder.config.enableActions = false;
    expect(scalableService(web)).toBeUndefined();

    holder.config.enableActions = true;
    holder.config.mode = "k8s";
    expect(scalableService(web)).toBeUndefined();

    holder.config.mode = "server";
    expect(scalableService(service("shop_web"))).toBeUndefined();
    expect(scalableService(undefined)).toBeUndefined();
  });
});
