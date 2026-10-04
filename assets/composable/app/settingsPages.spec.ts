import { describe, expect, test } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import { defineComponent } from "vue";

import { guardSettingsRoutes, settingsPages, settingsRedirect, type SettingsPageConfig } from "./settingsPages";

const server: SettingsPageConfig = { mode: "server", enableCloud: true, canLinkCloud: true };

describe("settingsPages", () => {
  test("server mode with cloud shows every page, preferences first and About last", () => {
    expect(settingsPages(server)).toEqual(["general", "logs", "sidebar", "updates", "cloud", "setup", "about"]);
  });

  test("swarm has no setup and no self-update", () => {
    expect(settingsPages({ ...server, mode: "swarm" })).toEqual(["general", "logs", "sidebar", "cloud", "about"]);
  });

  test("kubernetes shows nothing about updates", () => {
    expect(settingsPages({ ...server, mode: "k8s" })).not.toContain("updates");
  });

  test("cloud needs both the feature and the right to link", () => {
    expect(settingsPages({ ...server, canLinkCloud: false })).not.toContain("cloud");
    expect(settingsPages({ ...server, enableCloud: false })).not.toContain("cloud");
  });
});

describe("settingsRedirect", () => {
  const all = settingsPages(server);
  const swarm = settingsPages({ ...server, mode: "swarm" });

  test("/settings opens the first page", () => {
    expect(settingsRedirect("/settings", "", all)).toBe("/settings/general");
    expect(settingsRedirect("/settings/", "", all)).toBe("/settings/general");
  });

  test.each([
    ["#about", "/settings/about"],
    ["#appearance", "/settings/general"],
    ["#behavior", "/settings/general"],
    ["#logs", "/settings/logs"],
    ["#sidebar", "/settings/sidebar"],
    ["#setup", "/settings/setup"],
    ["#cloud", "/settings/cloud"],
  ])("old anchor %s lands on %s", (hash, target) => {
    expect(settingsRedirect("/settings", hash, all)).toBe(target);
  });

  test("an old anchor for a page this install hides opens the first page", () => {
    expect(settingsRedirect("/settings", "#setup", swarm)).toBe("/settings/general");
  });

  test("an unknown anchor opens the first page", () => {
    expect(settingsRedirect("/settings", "#nope", all)).toBe("/settings/general");
  });

  test("a hidden page falls back to the first page", () => {
    expect(settingsRedirect("/settings/updates", "", swarm)).toBe("/settings/general");
    expect(settingsRedirect("/settings/setup/", "", swarm)).toBe("/settings/general");
  });

  test("a visible page, an unknown page and other routes pass through", () => {
    expect(settingsRedirect("/settings/logs", "", all)).toBeUndefined();
    expect(settingsRedirect("/settings/nope", "", all)).toBeUndefined();
    expect(settingsRedirect("/settingsx", "", all)).toBeUndefined();
    expect(settingsRedirect("/", "#cloud", all)).toBeUndefined();
  });
});

describe("guardSettingsRoutes", () => {
  const Blank = defineComponent({ template: "<div/>" });

  function makeRouter(cfg: SettingsPageConfig = server) {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: "/", name: "/", component: Blank },
        {
          path: "/settings",
          name: "/settings",
          component: Blank,
          children: ["general", "logs", "sidebar", "updates", "cloud", "setup", "about"].map((id) => ({
            path: id,
            name: `/settings/${id}`,
            component: Blank,
          })),
        },
      ],
    });
    guardSettingsRoutes(router, settingsPages(cfg));
    return router;
  }

  test("the settings button's named route lands on the first page", async () => {
    const router = makeRouter();
    await router.push({ name: "/settings" });
    expect(router.currentRoute.value.fullPath).toBe("/settings/general");
  });

  test("an old hash link lands on its page without the hash", async () => {
    const router = makeRouter();
    await router.push("/settings#cloud");
    expect(router.currentRoute.value.fullPath).toBe("/settings/cloud");
  });

  test("the query survives the redirect", async () => {
    const router = makeRouter();
    await router.push("/settings?from=palette");
    expect(router.currentRoute.value.fullPath).toBe("/settings/general?from=palette");
  });

  test("a page this install hides is not rendered empty", async () => {
    const router = makeRouter({ ...server, mode: "k8s" });
    await router.push("/settings/updates");
    expect(router.currentRoute.value.fullPath).toBe("/settings/general");
  });

  test("a visible page is left alone", async () => {
    const router = makeRouter();
    await router.push("/settings/about");
    expect(router.currentRoute.value.fullPath).toBe("/settings/about");
  });
});
