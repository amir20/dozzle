import { Container } from "@/models/Container";

export type K8sWorkload = { namespace: string; kind: string; name: string };

// The kinds `kubectl rollout restart` supports, and so the backend.
const ROLLOUT_KINDS = new Set(["Deployment", "StatefulSet", "DaemonSet"]);

export function rolloutWorkload(namespace: string | undefined, kind: string | undefined, name: string | undefined) {
  if (config.mode !== "k8s" || !config.enableActions) return undefined;
  if (!namespace || !kind || !name || !ROLLOUT_KINDS.has(kind)) return undefined;
  return { namespace, kind, name } satisfies K8sWorkload;
}

// The outermost owner that can be rolled out: a Deployment rather than its
// ReplicaSet, and the StatefulSet under an operator's custom resource, which
// is the top of the chain but cannot be restarted itself.
export function containerWorkload(container: Container) {
  const owner = getK8sOwnerRefs(container).findLast((ref) => ROLLOUT_KINDS.has(ref.kind));
  return rolloutWorkload(container.labels["@k8s.namespace"], owner?.kind, owner?.name);
}

export const useRolloutRestart = () => {
  const { showToast } = useToast();
  const { t } = useI18n();
  const restarting = ref(false);

  async function rolloutRestart(workload: K8sWorkload) {
    const { namespace, kind, name } = workload;
    const url = `/api/k8s/workloads/${encodeURIComponent(namespace)}/${kind}/${encodeURIComponent(name)}/restart`;
    const label = `${kind}/${name}`;

    restarting.value = true;
    try {
      const response = await fetch(withBase(url), { method: "POST" });
      // Toast messages render as HTML, so anything from the cluster is escaped.
      if (response.ok) {
        showToast(
          {
            type: "info",
            title: t("toolbar.rollout-restart", { kind }),
            message: t("toolbar.rollout-restart-done", { workload: escapeHtml(label) }),
          },
          { expire: 5000 },
        );
      } else {
        // The API server's reason (RBAC, missing workload) is more useful than a generic line.
        const reason = (await response.text()).trim();
        const message = reason ? escapeHtml(reason) : t("error.unable-to-complete-action");
        showToast({ type: "error", title: t("error.action-failed"), message });
      }
    } catch {
      showToast({ type: "error", title: t("error.action-failed"), message: t("error.something-went-wrong") });
    } finally {
      restarting.value = false;
    }
  }

  return { restarting: readonly(restarting), rolloutRestart };
};
