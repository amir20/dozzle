import { Container } from "@/models/Container";

type ContainerActions = "start" | "stop" | "restart";
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

  async function update() {
    const updateUrl = `/api/hosts/${container.value.host}/containers/${container.value.id}/actions/update`;
    const toastId = "container-update";
    let reader: ReadableStreamDefaultReader<Uint8Array> | undefined;

    // Docker reports progress per layer, and layers keep appearing as the pull
    // goes on. Summing what is known so far gives a percentage that only ever
    // moves forward within a layer, which is close enough to be useful.
    const layers = new Map<string, { current: number; total: number }>();
    const pullProgress = () => {
      let current = 0;
      let total = 0;
      for (const layer of layers.values()) {
        current += layer.current;
        total += layer.total;
      }
      return total > 0 ? Math.min(100, (current / total) * 100) : undefined;
    };

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

      reader = response.body?.getReader();
      if (!reader) return;

      const decoder = new TextDecoder();
      let buffer = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n\n");
        buffer = lines.pop() ?? "";

        for (const chunk of lines) {
          const dataLine = chunk.split("\n").find((l) => l.startsWith("data: "));
          if (!dataLine) continue;

          const data = JSON.parse(dataLine.slice(6));

          switch (data.status) {
            case "pulling":
              if (data.layer && data.total > 0) {
                layers.set(data.layer, { current: data.current ?? 0, total: data.total });
                updateToast(toastId, { progress: pullProgress() });
              }
              break;
            case "recreating":
              // The pull is done; recreating cannot report progress, so the
              // bar goes away rather than sitting at an arbitrary value.
              updateToast(toastId, { message: t("toolbar.update-recreating"), progress: undefined });
              break;
            case "done":
            case "up-to-date":
              removeToast(toastId);
              showToast(
                {
                  title: t("toolbar.update"),
                  message: t(`toolbar.update-${data.status}`),
                  type: "info",
                },
                { expire: 3000 },
              );
              break;
            case "error":
              removeToast(toastId);
              showToast({
                type: "error",
                message: data.error || t("error.unknown-error"),
                title: t("error.update-failed"),
              });
              break;
          }
        }
      }
    } catch (error) {
      removeToast(toastId);
      showToast({ type: "error", message: t("error.something-went-wrong"), title: t("error.update-failed") });
    } finally {
      reader?.cancel();
      actionStates.update = false;
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
