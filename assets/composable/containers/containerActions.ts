import { Container } from "@/models/Container";

type ContainerActions = "start" | "stop" | "restart";

// One event of the update-progress stream, mirroring container.UpdateProgress.
export interface UpdateProgress {
  status: "pulling" | "recreating" | "done" | "up-to-date" | "error";
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

  // `self` is Dozzle's own container. "done" there means a helper container was
  // launched to replace it, so the page waits for the new one to answer instead
  // of reporting success while the old process is about to go away.
  async function update({ self = false }: { self?: boolean } = {}) {
    const updateUrl = `/api/hosts/${container.value.host}/containers/${container.value.id}/actions/update`;
    const toastId = "container-update";
    const pullProgress = createPullProgress();
    let restarting = false;

    actionStates.update = true;

    showToast(
      {
        id: toastId,
        title: t("toolbar.update"),
        message: t("toolbar.update-pulling"),
        type: "info",
      },
      { once: true },
    );

    try {
      const response = await fetch(withBase(updateUrl), { method: "POST" });
      if (!response.ok) {
        removeToast(toastId);
        showToast({ type: "error", message: t("error.unable-to-update"), title: t("error.update-failed") });
        return;
      }

      await readUpdateProgress(response, (data) => {
        if (data.status === "pulling") {
          if (data.layer && data.total && data.total > 0) {
            updateToast(toastId, { progress: pullProgress(data) });
          }
        } else if (data.status === "recreating") {
          // The pull is done; recreating cannot report progress, so the
          // bar goes away rather than sitting at an arbitrary value.
          updateToast(toastId, { message: t("toolbar.update-recreating"), progress: undefined });
        } else if (data.status === "done" && self) {
          restarting = true;
          updateToast(toastId, { message: t("setup.update.restarting"), progress: undefined });
        } else if (data.status === "done" || data.status === "up-to-date") {
          removeToast(toastId);
          showToast(
            {
              title: t("toolbar.update"),
              message: t(`toolbar.update-${data.status}`),
              type: "info",
            },
            { expire: 3000 },
          );
        } else if (data.status === "error") {
          removeToast(toastId);
          showToast({
            type: "error",
            message: data.error || t("error.unknown-error"),
            title: t("error.update-failed"),
          });
        }
      });
    } catch (error) {
      removeToast(toastId);
      showToast({ type: "error", message: t("error.something-went-wrong"), title: t("error.update-failed") });
      restarting = false;
    } finally {
      actionStates.update = false;
    }

    // The helper still has to start, rename and stop this container, so the old
    // process keeps answering for a while: only a reply after it went away counts.
    if (restarting && !(await useSetup().waitForRestart({ timeout: 180_000, mustGoDown: true }))) {
      removeToast(toastId);
      showToast({ type: "error", message: t("setup.restart.slow-body"), title: t("error.update-failed") });
    }
  }

  return {
    actionStates,
    start: () => actionHandler("start"),
    stop: () => actionHandler("stop"),
    restart: () => actionHandler("restart"),
    update,
  };
};
