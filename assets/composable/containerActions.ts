import { Container } from "@/models/Container";

type ContainerActions = "start" | "stop" | "restart";
export const useContainerActions = (container: Ref<Container>) => {
  const { showToast } = useToast();

  const actionStates = reactive({
    stop: false,
    restart: false,
    start: false,
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

  return {
    actionStates,
    start: () => actionHandler("start"),
    stop: () => actionHandler("stop"),
    restart: () => actionHandler("restart"),
  };
};
