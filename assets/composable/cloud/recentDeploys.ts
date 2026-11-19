import type { ContainerUpdate, DeployVerdict } from "@/models/ContainerUpdate";
import { useCloudConfig } from "./cloudConfig";

/** One update for the megaphone, with Dozzle Cloud's verdict when it has one. */
export type RecentDeploy = ContainerUpdate & { verdict?: DeployVerdict };

// Matches updateFeedLimit in internal/web/update_feed.go.
const MAX_RECENT = 20;

// The containers an update moved between name it on both sides. `at` would not:
// Cloud stores it to the microsecond, rounded, so it can come back a millisecond
// off. A swap Dozzle undid has the same old and new id, which still differs from
// the update that created that container.
const key = (u: ContainerUpdate) => `${u.host}/${u.oldId}/${u.newId}`;

/**
 * The updates Dozzle holds in memory, plus the ones only Dozzle Cloud still
 * remembers (made before a restart), newest first. A record both have keeps
 * Dozzle's fields, which carry the repo in their digests, and Cloud's verdict.
 * The key only matches Cloud against Dozzle: two swaps undone onto the same
 * container share one, and Dozzle's own records are never deduped by it.
 */
export function mergeRecentDeploys(local: ContainerUpdate[], cloud: RecentDeploy[]): RecentDeploy[] {
  const verdicts = new Map(cloud.map((d) => [key(d), d.verdict]));
  const known = new Set(local.map(key));
  const merged: RecentDeploy[] = [
    ...cloud.filter((d) => !known.has(key(d))),
    ...local.map((u) => ({ ...u, verdict: verdicts.get(key(u)) })),
  ];
  return merged.sort((a, b) => Date.parse(b.at) - Date.parse(a.at)).slice(0, MAX_RECENT);
}

// A Swarm task's container is "service.slot.taskid"; Cloud keys it without the
// task id (containers.ShortName), which is also what survives a redeploy.
const SWARM_TASK = /^(.+\.[^.]+)\.[a-z0-9]{25}$/;

/** Whether a deploy, keyed by host and the name Cloud stores, is this container's. */
export function deployOf(deploy: RecentDeploy, container: { host: string; name: string }): boolean {
  if (deploy.host !== container.host) return false;
  return deploy.name === container.name || deploy.name === container.name.replace(SWARM_TASK, "$1");
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
