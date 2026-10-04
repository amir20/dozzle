import type { Router } from "vue-router";
import type { Config } from "@/stores/config";

// Settings is one route per page under /settings. Which pages an install shows is a
// pure function of the injected config, so the nav and the route guard agree and both
// are testable without a browser.
export type SettingsPageId = "general" | "logs" | "sidebar" | "updates" | "cloud" | "setup" | "about";

export const SETTINGS_PAGE_IDS: readonly SettingsPageId[] = [
  "general",
  "logs",
  "sidebar",
  "updates",
  "cloud",
  "setup",
  "about",
];

export type SettingsPageConfig = Pick<Config, "mode" | "enableCloud" | "canLinkCloud">;

export function settingsPages(cfg: SettingsPageConfig): SettingsPageId[] {
  // Setup and the self-update schedule only exist in server mode, so swarm and
  // Kubernetes get neither page.
  const server = cfg.mode === "server";
  const visible: Record<SettingsPageId, boolean> = {
    general: true,
    logs: true,
    sidebar: true,
    updates: server,
    cloud: Boolean(cfg.enableCloud && cfg.canLinkCloud),
    setup: server,
    about: true,
  };
  return SETTINGS_PAGE_IDS.filter((id) => visible[id]);
}

// Settings used to be one scrolling page with an anchor per section. Old links and
// bookmarks still carry those anchors, so each one lands on the page that now holds it.
const LEGACY_HASHES: Record<string, SettingsPageId> = {
  "#about": "about",
  "#appearance": "general",
  "#behavior": "general",
  "#logs": "logs",
  "#sidebar": "sidebar",
  "#setup": "setup",
  "#cloud": "cloud",
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
  if (match && isPageId(match[1]) && !pages.includes(match[1])) return `/settings/${pages[0]}`;
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
