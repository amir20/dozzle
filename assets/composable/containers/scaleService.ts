import type { Service } from "@/models/Stack";

const TASK_NAME = "com.docker.swarm.task.name";

// A replicated service's tasks are named service.slot.id, a global service's
// service.nodeid.id. Only the first kind has a replica count to change, but a
// global one is still offered, so the dialog can say why rather than the menu
// row going missing without a word.
export function scalableService(service: Service | undefined) {
  if (!service || config.mode === "k8s" || !config.enableActions) return undefined;
  const task = service.containers.find((c) => c.labels[TASK_NAME])?.labels[TASK_NAME];
  if (!task?.startsWith(`${service.name}.`)) return undefined;
  const slot = task.slice(service.name.length + 1).split(".")[0];
  return { service, global: !/^\d+$/.test(slot) };
}

export const useScaleService = () => {
  const { showToast } = useToast();
  const { t } = useI18n();
  const scaling = ref(false);

  async function scaleService(name: string, replicas: number) {
    scaling.value = true;
    try {
      const response = await fetch(withBase(`/api/swarm/services/${encodeURIComponent(name)}/scale`), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ replicas }),
      });
      // Toast messages render as HTML, so anything from the engine is escaped.
      if (response.ok) {
        showToast(
          {
            type: "info",
            title: t("toolbar.scale-service"),
            message: t("toolbar.scale-done", { service: escapeHtml(name), count: replicas }, replicas),
          },
          { expire: 5000 },
        );
        return true;
      }
      // The engine's reason (no manager reachable, a concurrent update) beats a generic line.
      const reason = (await response.text()).trim();
      showToast({
        type: "error",
        title: t("error.action-failed"),
        message: reason ? escapeHtml(reason) : t("error.unable-to-complete-action"),
      });
    } catch {
      showToast({ type: "error", title: t("error.action-failed"), message: t("error.something-went-wrong") });
    } finally {
      scaling.value = false;
    }
    return false;
  }

  return { scaling: readonly(scaling), scaleService };
};
