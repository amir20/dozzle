import { Container } from "@/models/Container";
import { markRollingBack } from "./rollback";

type ContainerActions = "start" | "stop" | "restart";

// One event of the update-progress stream, mirroring container.UpdateProgress.
export interface UpdateProgress {
  // verifying: the new container started and is being watched to stay up.
  // rolled-back: it did not, and the previous container was put back.
  status: "pulling" | "recreating" | "verifying" | "done" | "up-to-date" | "rolled-back" | "error";
  layer?: string;
  current?: number;
  total?: number;
  error?: string;
}

// Docker reports progress per layer, and layers keep appearing as the pull
// goes on. Summing what is known so far gives a percentage that only ever
// moves forward within a layer, which is close enough to be useful.
export function createPullProgress() {
  const layers = new Map<string, { current: number; total: number }>();
  // Returns the overall percentage, or undefined while nothing has a known size.
  return function add(event: UpdateProgress) {
    if (event.layer && event.total && event.total > 0) {
      layers.set(event.layer, { current: event.current ?? 0, total: event.total });
    }
    let current = 0;
    let total = 0;
    for (const layer of layers.values()) {
      current += layer.current;
      total += layer.total;
    }
    return total > 0 ? Math.min(100, (current / total) * 100) : undefined;
  };
}

// Reads the text/event-stream both update endpoints answer with
// (/containers/{id}/actions/update and /api/update/self).
export async function readUpdateProgress(response: Response, onEvent: (event: UpdateProgress) => void) {
  const reader = response.body?.getReader();
  if (!reader) return;

  const decoder = new TextDecoder();
  let buffer = "";
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const chunks = buffer.split("\n\n");
      buffer = chunks.pop() ?? "";

      for (const chunk of chunks) {
        const dataLine = chunk.split("\n").find((l) => l.startsWith("data: "));
        if (!dataLine) continue;
        onEvent(JSON.parse(dataLine.slice(6)) as UpdateProgress);
      }
    }
  } finally {
    reader.cancel();
  }
}

export const useContainerActions = (container: Ref<Container>) => {
  const { showToast, updateToast, removeToast } = useToast();
  const { t } = useI18n();

  const actionStates = reactive({
    stop: false,
    restart: false,
    start: false,
    update: false,
    rollback: false,
  });

  async function actionHandler(action: ContainerActions) {
    const actionUrl = `/api/hosts/${container.value.host}/containers/${container.value.id}/actions/${action}`;

    const errors = {
      404: t("error.container-not-found"),
      500: t("error.unable-to-complete-action"),
      400: t("error.invalid-action"),
    } as Record<number, string>;

    const defaultError = t("error.something-went-wrong");
    const toastTitle = t("error.action-failed");

    actionStates[action] = true;

    try {
      const response = await fetch(withBase(actionUrl), { method: "POST" });
      if (!response.ok) {
        const message = errors[response.status] ?? defaultError;
        showToast({ type: "error", message, title: toastTitle });
      }
    } catch (error) {
      showToast({ type: "error", message: defaultError, title: toastTitle });
    }

    actionStates[action] = false;
  }

  // The words one streamed action (an update or a rollback) shows at each step.
  interface ProgressCopy {
    title: string;
    failed: string;
    unable: string;
    pulling: string;
    recreating: string;
    verifying: string;
    rolledBack: string;
    done: string;
    upToDate: string;
  }

  // Runs one streamed action and reports its progress in a single toast.
  // Returns whether it ended on "done".
  // `body`, when given, is sent as JSON.
  async function streamAction(url: string, toastId: string, copy: ProgressCopy, self: boolean, body?: object) {
    const pullProgress = createPullProgress();
    let restarting = false;
    let done = false;

    showToast({ id: toastId, title: copy.title, message: copy.pulling, type: "info" }, { once: true });

    try {
      const response = await fetch(
        withBase(url),
        body
          ? { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }
          : { method: "POST" },
      );
      if (!response.ok) {
        removeToast(toastId);
        showToast({ type: "error", message: copy.unable, title: copy.failed });
        return false;
      }

      await readUpdateProgress(response, (data) => {
        if (data.status === "pulling") {
          if (data.layer && data.total && data.total > 0) {
            updateToast(toastId, { progress: pullProgress(data) });
          }
        } else if (data.status === "recreating") {
          // The pull is done; recreating cannot report progress, so the
          // bar goes away rather than sitting at an arbitrary value.
          updateToast(toastId, { message: copy.recreating, progress: undefined });
        } else if (data.status === "verifying") {
          updateToast(toastId, { message: copy.verifying, progress: undefined });
        } else if (data.status === "rolled-back") {
          removeToast(toastId);
          showToast({
            type: "warning",
            // Same as an error: the reason quotes the engine's own text.
            message: copy.rolledBack + (data.error ? `<br>${escapeHtml(data.error)}` : ""),
            title: copy.failed,
          });
        } else if (data.status === "done" && self) {
          restarting = true;
          updateToast(toastId, { message: t("setup.update.restarting"), progress: undefined });
        } else if (data.status === "done" || data.status === "up-to-date") {
          done = data.status === "done";
          removeToast(toastId);
          showToast(
            { title: copy.title, message: data.status === "done" ? copy.done : copy.upToDate, type: "info" },
            { expire: 3000 },
          );
        } else if (data.status === "error") {
          removeToast(toastId);
          showToast({
            type: "error",
            // Toasts render HTML, and pull errors carry the registry's own text.
            message: data.error ? escapeHtml(data.error) : t("error.unknown-error"),
            title: copy.failed,
          });
        }
      });
    } catch (error) {
      removeToast(toastId);
      showToast({ type: "error", message: t("error.something-went-wrong"), title: copy.failed });
      restarting = false;
    }

    // The helper still has to start, rename and stop this container, so the old
    // process keeps answering for a while: only a reply after it went away counts.
    if (restarting && !(await useSetup().waitForRestart({ timeout: 180_000, mustGoDown: true }))) {
      removeToast(toastId);
      showToast({ type: "error", message: t("setup.restart.slow-body"), title: copy.failed });
    }
    return done;
  }

  // `self` is Dozzle's own container. "done" there means a helper container was
  // launched to replace it, so the page waits for the new one to answer instead
  // of reporting success while the old process is about to go away.
  // `watchInCloud` is "Have Dozzle Cloud watch this update"; left out, the
  // update is not watched.
  async function update({ self = false, watchInCloud }: { self?: boolean; watchInCloud?: boolean } = {}) {
    actionStates.update = true;
    try {
      await streamAction(
        `/api/hosts/${container.value.host}/containers/${container.value.id}/actions/update`,
        "container-update",
        {
          title: t("toolbar.update"),
          failed: t("error.update-failed"),
          unable: t("error.unable-to-update"),
          pulling: t("toolbar.update-pulling"),
          recreating: t("toolbar.update-recreating"),
          verifying: t("toolbar.update-verifying"),
          rolledBack: t("toolbar.update-rolled-back"),
          done: t("toolbar.update-done"),
          upToDate: t("toolbar.update-up-to-date"),
        },
        self,
        watchInCloud === undefined ? undefined : { watchInCloud },
      );
    } finally {
      actionStates.update = false;
    }
  }

  // Swaps the container back to `toImageId`, the target the UI offered. The
  // server refuses anything that is not this container's previous image.
  // The running flag is shared, since the dialog that starts a rollback is not
  // the toolbar or drawer row that offered it.
  async function rollback(toImageId: string) {
    const target = { host: container.value.host, id: container.value.id };
    actionStates.rollback = true;
    markRollingBack(target, true);
    try {
      return await streamAction(
        `/api/hosts/${container.value.host}/containers/${container.value.id}/actions/rollback?to=${encodeURIComponent(toImageId)}`,
        "container-rollback",
        {
          title: t("rollback.title-short"),
          failed: t("rollback.failed"),
          unable: t("rollback.unable"),
          pulling: t("rollback.pulling"),
          recreating: t("toolbar.update-recreating"),
          verifying: t("rollback.verifying"),
          rolledBack: t("rollback.undone"),
          done: t("rollback.done"),
          upToDate: t("rollback.done"),
        },
        false,
      );
    } finally {
      actionStates.rollback = false;
      markRollingBack(target, false);
    }
  }

  return {
    actionStates,
    start: () => actionHandler("start"),
    stop: () => actionHandler("stop"),
    restart: () => actionHandler("restart"),
    update,
    rollback,
  };
};
