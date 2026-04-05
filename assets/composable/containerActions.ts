import { Container } from "@/models/Container";

type ContainerActions = "start" | "stop" | "restart";
export const useContainerActions = (container: Ref<Container>) => {
  const { showToast, removeToast } = useToast();
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
      404: "container not found",
      500: "unable to complete action",
      400: "invalid action",
    } as Record<number, string>;

    const defaultError = "something went wrong";
    const toastTitle = "Action Failed";

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
        showToast({ type: "error", message: "unable to update container", title: "Update Failed" });
        actionStates.update = false;
        return;
      }

      const reader = response.body?.getReader();
      if (!reader) {
        removeToast(toastId);
        actionStates.update = false;
        return;
      }

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
              break;
            case "recreating":
              removeToast(toastId);
              showToast(
                {
                  id: toastId,
                  title: t("toolbar.update"),
                  message: t("toolbar.update-recreating"),
                  type: "info",
                },
                { once: true },
              );
              break;
            case "done":
              removeToast(toastId);
              showToast(
                {
                  title: t("toolbar.update"),
                  message: t("toolbar.update-done"),
                  type: "info",
                },
                { expire: 3000 },
              );
              break;
            case "up-to-date":
              removeToast(toastId);
              showToast(
                {
                  title: t("toolbar.update"),
                  message: t("toolbar.update-up-to-date"),
                  type: "info",
                },
                { expire: 3000 },
              );
              break;
            case "error":
              removeToast(toastId);
              showToast({
                type: "error",
                message: data.error || "unknown error",
                title: "Update Failed",
              });
              break;
          }
        }
      }
    } catch (error) {
      removeToast(toastId);
      showToast({ type: "error", message: "something went wrong", title: "Update Failed" });
    }

    actionStates.update = false;
  }

  return {
    actionStates,
    start: () => actionHandler("start"),
    stop: () => actionHandler("stop"),
    restart: () => actionHandler("restart"),
    update,
  };
};
