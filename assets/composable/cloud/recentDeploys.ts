import type { ContainerUpdate, DeployVerdict } from "@/models/ContainerUpdate";
import { useCloudConfig } from "./cloudConfig";

/** One update for the megaphone, with Dozzle Cloud's verdict when it has one. */
export type RecentDeploy = ContainerUpdate & { verdict?: DeployVerdict };

// Matches updateFeedLimit in internal/web/update_feed.go.
const MAX_RECENT = 20;

// Dozzle pushes `at` to Cloud in nanoseconds and gets it back the same, so the
// millisecond it parses to is a key both sides agree on.
const key = (u: ContainerUpdate) => `${u.host}/${u.newId}/${Date.parse(u.at)}`;

/**
 * The updates Dozzle holds in memory, plus the ones only Dozzle Cloud still
 * remembers (made before a restart), newest first. A record both have keeps
 * Dozzle's fields, which carry the repo in their digests, and Cloud's verdict.
 */
export function mergeRecentDeploys(local: ContainerUpdate[], cloud: RecentDeploy[]): RecentDeploy[] {
  const verdicts = new Map(cloud.map((d) => [key(d), d]));
  const merged = new Map<string, RecentDeploy>();
  for (const d of cloud) merged.set(key(d), d);
  for (const u of local) merged.set(key(u), { ...u, verdict: verdicts.get(key(u))?.verdict });
  return [...merged.values()].sort((a, b) => Date.parse(b.at) - Date.parse(a.at)).slice(0, MAX_RECENT);
}

const cloudDeploys = ref<RecentDeploy[]>([]);

/**
 * Dozzle Cloud's record of recent updates, fetched on demand. Only asks when
 * Cloud is linked, and never throws: on any failure the megaphone keeps showing
 * what Dozzle remembers.
 */
export function useRecentDeploys() {
  const { cloudConfig } = useCloudConfig();
  const linked = computed(() => !!cloudConfig.value?.linked);

  async function refresh() {
    // Unlinked since the last ask: what Cloud said no longer applies.
    if (!linked.value) {
      cloudDeploys.value = [];
      return;
    }
    try {
      const res = await fetch(withBase("/api/cloud/deploys/recent"));
      if (res.ok) cloudDeploys.value = (await res.json()) ?? [];
    } catch {
      // keep the last answer
    }
  }

  return { cloudDeploys, refresh };
}
