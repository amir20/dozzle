import { powerAction, type Container } from "@/models/Container";

export type BulkAction = "start" | "stop" | "restart";

export type BulkPlan = { action: BulkAction; targets: Container[]; skippedSelf: boolean };

// At most this many action requests are in flight at once, so a big stack does
// not hit the engine with dozens of restarts in the same instant.
export const BULK_CONCURRENCY = 4;

/**
 * Which containers of a view an action would touch. Restart and stop only take
 * running ones, a subset of what the single toolbar offers, so a paused or
 * deleted container is never acted on. Dozzle's own container is skipped for
 * both: stopping it takes the page down with it.
 */
export function bulkPlan(containers: Container[], action: BulkAction): BulkPlan {
  if (action === "start") {
    return { action, targets: containers.filter((c) => powerAction(c) === "start"), skippedSelf: false };
  }
  const running = containers.filter((c) => c.state === "running");
  const targets = running.filter((c) => !servesThisPage(c));
  const skippedSelf = targets.length < running.length;
  return { action, targets, skippedSelf };
}

// Unlike isSelf, which leaves swarm to the orchestrator for updates, a swarm task
// still takes the page down when it is stopped. Every swarm node reports as a
// "swarm" host, so the local-host check only applies outside swarm.
function servesThisPage(c: Container) {
  const self = config.selfContainerId;
  if (!self || c.id.length < 12 || !self.startsWith(c.id)) return false;
  return c.isSwarm || config.hosts.find((host) => host.id === c.host)?.type === "local";
}

/** Runs `task` over every item, `limit` at a time. Resolves to the items whose task failed. */
export async function runBounded<T>(items: T[], limit: number, task: (item: T) => Promise<boolean>) {
  const failed: T[] = [];
  let next = 0;
  async function worker() {
    while (next < items.length) {
      const item = items[next++];
      const ok = await task(item).catch(() => false);
      if (!ok) failed.push(item);
    }
  }
  await Promise.all(Array.from({ length: Math.min(limit, items.length) }, worker));
  return failed;
}

export function useBulkContainerActions(containers: MaybeRefOrGetter<Container[]>) {
  const { showToast } = useToast();
  const { t } = useI18n();

  // k8s has no stop or start for a pod, and restarting every pod at once is what the
  // workload's rollout restart already does without taking it down, so none here.
  const actions: BulkAction[] = config.mode === "k8s" ? [] : ["restart", "stop", "start"];

  /** The actions that would touch at least one container right now. */
  const available = computed(() =>
    actions.map((action) => bulkPlan(toValue(containers), action)).filter((plan) => plan.targets.length > 0),
  );

  const running = ref<BulkAction>();

  async function run(plan: BulkPlan) {
    if (running.value || plan.targets.length === 0) return;
    running.value = plan.action;
    try {
      const failed = await runBounded(plan.targets, BULK_CONCURRENCY, async (c) => {
        const url = `/api/hosts/${c.host}/containers/${c.id}/actions/${plan.action}`;
        const response = await fetch(withBase(url), { method: "POST" });
        return response.ok;
      });
      if (failed.length > 0) {
        showToast({
          type: "error",
          title: t("error.action-failed"),
          message: t("toolbar.bulk-failed", { names: escapeHtml(failed.map((c) => c.name).join(", ")) }),
        });
      }
    } finally {
      running.value = undefined;
    }
  }

  return { available, running: readonly(running), run };
}
