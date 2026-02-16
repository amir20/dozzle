import { Container } from "@/models/Container";
import type { ContainerJson } from "@/types/Container";
import type { Dispatcher, NotificationRule, PreviewResult } from "@/types/notifications";
import { createExprEditor, createContainerHints } from "@/composable/exprEditor";

export interface AlertFormOptions {
  close?: () => void;
  onCreated?: () => void;
  alert?: NotificationRule;
  prefill?: { name?: string; containerExpression?: string; logExpression?: string; metricExpression?: string };
}

export interface ContainerResult {
  error?: string;
  containers?: Container[];
}

export function useAlertForm(options: AlertFormOptions) {
  const isEditing = computed(() => !!options.alert);
  const alertName = ref(options.alert?.name ?? options.prefill?.name ?? "");
  const containerExpression = ref(options.alert?.containerExpression ?? options.prefill?.containerExpression ?? "");
  const dispatcherId = ref(options.alert?.dispatcher?.id ?? 0);
  const isSaving = ref(false);
  const saveError = ref<string | null>(null);

  // Destinations
  const destinations = ref<Dispatcher[]>([]);
  onMounted(async () => {
    const res = await fetch(withBase("/api/notifications/dispatchers"));
    destinations.value = await res.json();
  });
  const selectedDestination = computed(() => destinations.value.find((d) => d.id === dispatcherId.value));

  // Container store for autocomplete
  const containerStore = useContainerStore();
  const { containers } = storeToRefs(containerStore);
  const containerNames = computed(() => [
    ...new Set(containers.value.filter((c) => c.state === "running").map((c) => c.name)),
  ]);
  const imageNames = computed(() => [...new Set(containers.value.map((c) => c.image))]);
  const hostNames = computed(() => [...new Set(containers.value.map((c) => c.host))]);

  // Container validation
  const containerResult = ref<ContainerResult | null>(null);
  const isLoading = ref(false);

  const baseCanSave = computed(
    () =>
      alertName.value.trim() &&
      containerExpression.value.trim() &&
      dispatcherId.value > 0 &&
      !containerResult.value?.error &&
      !isSaving.value,
  );

  async function initContainerEditor(el: HTMLElement) {
    return await createExprEditor({
      parent: el,
      placeholder: 'name contains "api"',
      initialValue: options.alert?.containerExpression ?? options.prefill?.containerExpression ?? "",
      getHints: () => createContainerHints(containerNames.value, imageNames.value, hostNames.value),
      onChange: (v) => (containerExpression.value = v),
    });
  }

  async function saveAlert(typeSpecificFields: Record<string, unknown>) {
    isSaving.value = true;
    saveError.value = null;
    try {
      const input = {
        name: alertName.value.trim(),
        containerExpression: containerExpression.value,
        dispatcherId: dispatcherId.value,
        enabled: true,
        ...typeSpecificFields,
      };
      const url = isEditing.value
        ? withBase(`/api/notifications/rules/${options.alert!.id}`)
        : withBase("/api/notifications/rules");
      const res = await fetch(url, {
        method: isEditing.value ? "PUT" : "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(input),
      });
      if (!res.ok) {
        const data = await res.json();
        throw new Error(data.error || "Failed to save alert");
      }
      options.onCreated?.();
      options.close?.();
    } catch (e) {
      saveError.value = e instanceof Error ? e.message : "Failed to save alert";
    } finally {
      isSaving.value = false;
    }
  }

  async function validatePreview(extraFields: Record<string, unknown> = {}) {
    if (!containerExpression.value && !Object.values(extraFields).some(Boolean)) {
      containerResult.value = null;
      return { data: null };
    }

    isLoading.value = true;
    try {
      const res = await fetch(withBase("/api/notifications/preview"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          containerExpression: containerExpression.value,
          ...extraFields,
        }),
      });
      if (!res.ok) {
        const errData = await res.json();
        throw new Error(errData.error || "Preview failed");
      }
      const data: PreviewResult = await res.json();
      containerResult.value = containerExpression.value
        ? {
            error: data.containerError ?? undefined,
            containers: data.matchedContainers?.map((c) => Container.fromJSON(c as ContainerJson)),
          }
        : null;
      return { data };
    } catch (e) {
      containerResult.value = { error: e instanceof Error ? e.message : "Unknown error" };
      return { data: null };
    } finally {
      isLoading.value = false;
    }
  }

  return {
    isEditing,
    alertName,
    containerExpression,
    dispatcherId,
    destinations,
    selectedDestination,
    containerResult,
    isLoading,
    isSaving,
    saveError,
    baseCanSave,
    initContainerEditor,
    saveAlert,
    validatePreview,
  };
}
