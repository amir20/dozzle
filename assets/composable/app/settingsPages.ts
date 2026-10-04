import type { Router } from "vue-router";
import type { Config } from "@/stores/config";

// Settings is one route per page under /settings. Which pages an install shows is a
// pure function of the injected config, so the nav and the route guard agree and both
// are testable without a browser.
export type SettingsPageId = "general" | "logs" | "sidebar" | "security" | "hosts" | "updates" | "cloud" | "about";

export const SETTINGS_PAGE_IDS: readonly SettingsPageId[] = [
  "general",
  "logs",
  "sidebar",
  "security",
  "hosts",
  "updates",
  "cloud",
  "about",
];

// The pages that describe the server rather than this browser. The first one opens
// the nav's second group.
export const SERVER_PAGES: readonly SettingsPageId[] = ["security", "hosts", "updates", "cloud"];

export type SettingsPageConfig = Pick<Config, "mode" | "enableCloud" | "canLinkCloud">;

export function settingsPages(cfg: SettingsPageConfig): SettingsPageId[] {
  // The self-update schedule only exists in server mode. Security and Hosts show in
  // every mode: swarm and Kubernetes have no setup API, so there they only report
  // what flags and the orchestrator decided.
  const server = cfg.mode === "server";
  const visible: Record<SettingsPageId, boolean> = {
    general: true,
    logs: true,
    sidebar: true,
    security: true,
    hosts: true,
    updates: server,
    cloud: Boolean(cfg.enableCloud && cfg.canLinkCloud),
    about: true,
  };
  return SETTINGS_PAGE_IDS.filter((id) => visible[id]);
}

// The pages that hold this browser's preferences, the ones "Reset to defaults" puts back.
// The rest describe the server or this install, so a reset there would change settings
// the page does not show.
export const PREFERENCE_PAGES: readonly SettingsPageId[] = ["general", "logs", "sidebar"];

export const isPreferencePage = (path: string): boolean => {
  const match = /^\/settings\/([^/]+)\/?$/.exec(path);
  return Boolean(match && (PREFERENCE_PAGES as readonly string[]).includes(match[1]));
};

// Moving from one settings page to another opens the new page at its top. The nav is
// sticky, so without this a click after scrolling down a long page lands mid-way into
// the next one with its title off screen.
export const isSettingsPageSwitch = (to: string, from: string): boolean =>
  to !== from && /^\/settings\/[^/]+\/?$/.test(to) && /^\/settings(\/|$)/.test(from);

// Settings used to be one scrolling page with an anchor per section. Old links and
// bookmarks still carry those anchors, so each one lands on the page that now holds it.
const LEGACY_HASHES: Record<string, SettingsPageId> = {
  "#about": "about",
  "#appearance": "general",
  "#behavior": "general",
  "#logs": "logs",
  "#sidebar": "sidebar",
  // The Setup card that reopened the wizard is "Run setup again" on About now.
  "#setup": "about",
  "#cloud": "cloud",
};

// Pages that no longer exist, and the page that took over what they held.
const MOVED_PAGES: Record<string, SettingsPageId> = {
  setup: "about",
};

const isPageId = (id: string): id is SettingsPageId => (SETTINGS_PAGE_IDS as readonly string[]).includes(id);

// Where a navigation into /settings should go instead, or undefined to let it through.
// /settings itself opens the first page (or the one an old anchor names), and a page
// this install does not show falls back to the first one rather than rendering empty.
export function settingsRedirect(path: string, hash: string, pages: SettingsPageId[]): string | undefined {
  const trimmed = path.length > 1 ? path.replace(/\/+$/, "") : path;
  if (trimmed === "/settings") {
    const fromHash = LEGACY_HASHES[hash];
    const target = fromHash && pages.includes(fromHash) ? fromHash : pages[0];
    return `/settings/${target}`;
  }
  const match = /^\/settings\/([^/]+)$/.exec(trimmed);
  if (!match) return undefined;
  const moved = Object.hasOwn(MOVED_PAGES, match[1]) ? MOVED_PAGES[match[1]] : undefined;
  if (moved) return `/settings/${pages.includes(moved) ? moved : pages[0]}`;
  if (isPageId(match[1]) && !pages.includes(match[1])) return `/settings/${pages[0]}`;
  return undefined;
}

// Runs before the route resolves, so /settings never renders an empty layout and an
// old anchor never flashes the wrong page first.
export function guardSettingsRoutes(router: Router, pages: SettingsPageId[]) {
  return router.beforeEach((to) => {
    const target = settingsRedirect(to.path, to.hash, pages);
    return target ? { path: target, query: to.query, replace: true } : undefined;
  });
}
