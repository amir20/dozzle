/**
 * @vitest-environment jsdom
 */
import { describe, expect, test, vi } from "vitest";
import { createRouter, createMemoryHistory } from "vue-router";
import { prefetchRoutes } from "./router";

const page = () => ({ template: "<div/>" });

describe("prefetchRoutes", () => {
  test("imports every lazily loaded page", async () => {
    const home = vi.fn().mockResolvedValue(page());
    const settings = vi.fn().mockResolvedValue(page());
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: "/", component: home },
        { path: "/settings", component: settings },
      ],
    });

    await prefetchRoutes(router);

    expect(home).toHaveBeenCalledOnce();
    expect(settings).toHaveBeenCalledOnce();
  });

  test("leaves eagerly bundled pages alone and survives a failed chunk", async () => {
    const broken = vi.fn().mockRejectedValue(new Error("Failed to fetch dynamically imported module"));
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: "/", component: page() },
        { path: "/broken", component: broken },
      ],
    });

    await expect(prefetchRoutes(router)).resolves.toBeDefined();
    expect(broken).toHaveBeenCalledOnce();
  });
});
