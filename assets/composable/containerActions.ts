type ContainerActions = "start" | "stop" | "restart";
export const useContainerActions = () => {
  const context = inject(containerContext);
  if (!context) {
    throw new Error("No container context provided");
  }

  const container = computed(() => context.container.value);
  const { showToast } = useToast();

  const actionStates = reactive({
    stop: false,
    restart: false,
    start: false,
  });

  async function actionHandler(action: ContainerActions) {
    const actionUrl = `/api/actions/${action}/${container.value.host}/${container.value.id}`;
    const errors = {
      404: "container not found",
      500: "unable to complete action",
      400: "invalid action",
    } as Record<number, string>;
    const defaultError = "something went wrong";

    actionStates[`${action}`] = true;

    try {
      await fetch(withBase(actionUrl), { method: "POST" }).then((response) => {
        if (!response.ok) {
          showToast({ type: "error", message: errors[response.status] ?? defaultError, title: "Action failed" });
        }
      });
    } catch (error) {
      showToast({ type: "error", message: defaultError, title: "Container action failed!" });
    }

    actionStates[`${action}`] = false;
  }

  return {
    actionStates,
    start: () => actionHandler("start"),
    stop: () => actionHandler("stop"),
    restart: () => actionHandler("restart"),
  };
};
