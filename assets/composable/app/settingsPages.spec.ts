import { describe, expect, test } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import { defineComponent } from "vue";

import {
  guardSettingsRoutes,
  isPreferencePage,
  isSettingsPageSwitch,
  settingsPages,
  settingsRedirect,
  type SettingsPageConfig,
} from "./settingsPages";

const server: SettingsPageConfig = { mode: "server", enableCloud: true, canLinkCloud: true };

describe("settingsPages", () => {
  test("server mode with cloud shows every page, preferences first and About last", () => {
    expect(settingsPages(server)).toEqual([
      "general",
      "logs",
      "sidebar",
      "security",
      "hosts",
      "updates",
      "cloud",
      "about",
    ]);
  });

  test("swarm keeps Security and Hosts as read-only status, without self-update", () => {
    expect(settingsPages({ ...server, mode: "swarm" })).toEqual([
      "general",
      "logs",
      "sidebar",
      "security",
      "hosts",
      "cloud",
      "about",
    ]);
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
    ["#setup", "/settings/about"],
    ["#cloud", "/settings/cloud"],
  ])("old anchor %s lands on %s", (hash, target) => {
    expect(settingsRedirect("/settings", hash, all)).toBe(target);
  });

  test("an old anchor for a page this install hides opens the first page", () => {
    expect(settingsRedirect("/settings", "#cloud", settingsPages({ ...server, enableCloud: false }))).toBe(
      "/settings/general",
    );
  });

  test("the old Setup page lands on About, where Run setup again lives", () => {
    expect(settingsRedirect("/settings/setup", "", all)).toBe("/settings/about");
    expect(settingsRedirect("/settings/setup/", "", swarm)).toBe("/settings/about");
  });

  test("a page name that collides with an object key is not a moved page", () => {
    expect(settingsRedirect("/settings/constructor", "", all)).toBeUndefined();
  });

  test("an unknown anchor opens the first page", () => {
    expect(settingsRedirect("/settings", "#nope", all)).toBe("/settings/general");
  });

  test("a hidden page falls back to the first page", () => {
    expect(settingsRedirect("/settings/updates", "", swarm)).toBe("/settings/general");
    expect(settingsRedirect("/settings/updates/", "", swarm)).toBe("/settings/general");
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
        // Like the app's [...all].vue, which is what a removed page used to fall into.
        { path: "/:all(.*)*", name: "/[...all]", component: Blank },
        {
          path: "/settings",
          name: "/settings",
          component: Blank,
          children: ["general", "logs", "sidebar", "security", "hosts", "updates", "cloud", "about"].map((id) => ({
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

  // The cases themselves are settingsRedirect's, above. This only checks the guard
  // wires it in: the hash is dropped and the query kept.
  test("redirects through settingsRedirect, keeping the query", async () => {
    const router = makeRouter();
    await router.push("/settings?from=palette#cloud");
    expect(router.currentRoute.value.fullPath).toBe("/settings/cloud?from=palette");
  });
});

describe("isPreferencePage", () => {
  test("reset belongs to the pages that show preferences", () => {
    expect(isPreferencePage("/settings/general")).toBe(true);
    expect(isPreferencePage("/settings/logs")).toBe(true);
    expect(isPreferencePage("/settings/sidebar/")).toBe(true);
  });

  test("server and install pages have nothing a reset would put back", () => {
    for (const id of ["security", "hosts", "updates", "cloud", "about"])
      expect(isPreferencePage(`/settings/${id}`)).toBe(false);
    expect(isPreferencePage("/settings")).toBe(false);
    expect(isPreferencePage("/container/abc")).toBe(false);
  });
});

describe("isSettingsPageSwitch", () => {
  test("another settings page opens at its top", () => {
    expect(isSettingsPageSwitch("/settings/sidebar", "/settings/logs")).toBe(true);
    expect(isSettingsPageSwitch("/settings/general", "/settings")).toBe(true);
  });

  test("the same page, or arriving from elsewhere, keeps the browser's behaviour", () => {
    expect(isSettingsPageSwitch("/settings/logs", "/settings/logs")).toBe(false);
    expect(isSettingsPageSwitch("/settings/logs", "/container/abc")).toBe(false);
    expect(isSettingsPageSwitch("/container/abc", "/settings/logs")).toBe(false);
    expect(isSettingsPageSwitch("/settingsx/a", "/settings/logs")).toBe(false);
  });
});
