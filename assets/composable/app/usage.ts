// Counts what only the browser sees (searches, the command palette, the wizard) and
// hands them to the server, which folds them into the daily usage beacon. Numbers
// only: a key is a fixed name, never anything the user typed or looked at. Kept in
// memory, never in storage, and off entirely under --no-analytics.

import { i18n } from "@/modules/i18n";

export type UsageKey =
  | "logs.search"
  | "logs.sql"
  | "palette.open"
  | "pinned.open"
  | "wizard.shown"
  | "wizard.finished"
  | "wizard.skip.login"
  | "wizard.skip.actions"
  | "wizard.skip.hosts"
  | "wizard.skip.cloud"
  | "wizard.skip.update"
  | "cloud.welcome"
  | "cloud.connect"
  | "stream.reconnect";

export const USAGE_FLUSH_INTERVAL = 5 * 60 * 1000;
const MINUTE = 60 * 1000;

interface UsageDeps {
  enabled: () => boolean;
  locale: () => string;
  post: (body: string) => void;
  visible: () => boolean;
}

// The batcher on its own, so it can be tested without a page.
export function createUsageBatcher(deps: UsageDeps) {
  const counts = new Map<UsageKey, number>();
  let activeMinutes = 0;
  let localeSent = false;

  function track(key: UsageKey) {
    if (!deps.enabled()) return;
    counts.set(key, (counts.get(key) ?? 0) + 1);
  }

  // Called once a minute; only a minute with the page in front counts.
  function tick() {
    if (deps.enabled() && deps.visible()) activeMinutes++;
  }

  function flush() {
    if (!deps.enabled()) return;
    if (counts.size === 0 && activeMinutes === 0 && localeSent) return;
    const body: { counts: Record<string, number>; activeMinutes: number; locale?: string } = {
      counts: Object.fromEntries(counts),
      activeMinutes,
    };
    // One session, one locale: sent with the first report only.
    if (!localeSent) body.locale = deps.locale();
    counts.clear();
    activeMinutes = 0;
    localeSent = true;
    deps.post(JSON.stringify(body));
  }

  return { track, tick, flush };
}

const batcher = createUsageBatcher({
  enabled: () => !config.noAnalytics,
  locale: () => String(i18n.global.locale.value),
  visible: () => document.visibilityState === "visible",
  post: (body) => {
    // keepalive so a report sent as the tab closes still goes out.
    fetch(withBase("/api/usage"), {
      method: "POST",
      body,
      headers: { "Content-Type": "application/json" },
      keepalive: true,
    }).catch(() => {});
  },
});

export function trackUsage(key: UsageKey) {
  batcher.track(key);
}

let started = false;

// Called once by the layout.
export function startUsageReporting() {
  if (started || config.noAnalytics) return;
  started = true;
  setInterval(batcher.tick, MINUTE);
  setInterval(batcher.flush, USAGE_FLUSH_INTERVAL);
  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "hidden") batcher.flush();
  });
}
