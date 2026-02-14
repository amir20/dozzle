<template>
  <div class="space-y-4 p-4">
    <div class="mb-6">
      <h2 class="text-2xl font-bold">
        {{ isEditing ? $t("notifications.alert-form.edit-title") : $t("notifications.alert-form.create-title") }}
      </h2>
      <p class="text-base-content/60">{{ $t("notifications.alert-form.description") }}</p>
    </div>

    <!-- Alert Name -->
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.alert-form.alert-name") }}</legend>
      <input
        ref="alertNameInput"
        v-model="alertName"
        type="text"
        class="input focus:input-primary w-full text-base"
        :class="alertName.trim() ? 'input-primary' : ''"
        required
        :placeholder="$t('notifications.alert-form.alert-name-placeholder')"
      />
    </fieldset>

    <!-- Container Filter -->
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.alert-form.container-filter") }}</legend>
      <div
        class="input focus-within:input-primary w-full focus-within:z-50"
        :class="
          containerExpression.trim() && !containerResult?.error
            ? 'input-primary'
            : { 'input-error!': containerResult?.error }
        "
      >
        <div ref="containerEditorRef" class="w-full"></div>
      </div>
      <div v-if="containerResult" class="fieldset-label">
        <span v-if="containerResult.error" class="text-error">{{ containerResult.error }}</span>
        <span v-else-if="containerResult.containers?.length" class="text-success">
          <mdi:check class="inline" />
          {{
            $t("notifications.alert-form.containers-match", {
              count: containerResult.containers.length,
              names: containerResult.containers.map((c) => c.name).join(", "),
            })
          }}
        </span>
        <span v-else class="text-warning">
          <mdi:alert class="inline" />
          {{ $t("notifications.alert-form.no-containers-match") }}
        </span>
      </div>
    </fieldset>

    <!-- Log Filter -->
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.alert-form.log-filter") }}</legend>
      <div
        class="input focus-within:input-primary w-full focus-within:z-50"
        :class="logExpression.trim() && !logError ? 'input-primary' : { 'input-error!': logError }"
      >
        <div ref="logEditorRef" class="w-full"></div>
      </div>
      <div v-if="logError || logExpression" class="fieldset-label">
        <span v-if="logError" class="text-error">{{ logError }}</span>
        <span v-else-if="logMessages.length" class="text-success">
          <mdi:check class="inline" />
          {{ $t("notifications.alert-form.logs-match", { count: logTotalCount }) }}
        </span>
        <span v-else-if="!isLoading" class="text-warning">
          <mdi:alert class="inline" />
          {{ $t("notifications.alert-form.no-logs-match") }}
        </span>
      </div>
    </fieldset>

    <!-- Destination -->
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.alert-form.destination") }}</legend>
      <details class="dropdown w-full" ref="destinationDropdown">
        <summary class="btn btn-outline w-full justify-between" :class="{ 'btn-primary': selectedDestination }">
          <span class="flex items-center gap-2">
            <template v-if="selectedDestination">
              <mdi:webhook v-if="selectedDestination.type === 'webhook'" />
              <mdi:cloud v-else />
              {{ selectedDestination.name }}
            </template>
            <span v-else class="text-base-content/60">{{ $t("notifications.alert-form.select-destination") }}</span>
          </span>
          <carbon:caret-down />
        </summary>
        <ul class="dropdown-content menu bg-base-200 rounded-box z-50 mt-1 w-full border p-2 shadow-sm">
          <li v-for="dest in destinations" :key="dest.id">
            <a
              @click="
                dispatcherId = dest.id;
                destinationDropdown?.removeAttribute('open');
              "
              :class="{ active: dispatcherId === dest.id }"
            >
              <mdi:webhook v-if="dest.type === 'webhook'" />
              <mdi:cloud v-else />
              {{ dest.name }}
            </a>
          </li>
        </ul>
      </details>
      <div v-if="!destinations.length" class="fieldset-label">
        <span class="text-warning">
          <mdi:alert class="inline" />
          {{ $t("notifications.alert-form.no-destinations") }}
        </span>
      </div>
    </fieldset>

    <!-- Log Preview -->
    <div v-if="logMessages.length" class="mt-4">
      <div class="mb-2 text-lg">{{ $t("notifications.alert-form.preview") }}</div>
      <LogList
        :messages="logMessages"
        :last-selected-item="undefined"
        class="border-base-content/50 h-64 overflow-hidden rounded-lg border"
      />
    </div>

    <!-- Error -->
    <div v-if="saveError" class="alert alert-error">
      <span>{{ saveError }}</span>
    </div>

    <!-- Actions -->
    <div class="flex justify-end gap-2 pt-4">
      <button class="btn" @click="close?.()">{{ $t("notifications.alert-form.cancel") }}</button>
      <button class="btn btn-primary" :disabled="!canSave" @click="saveAlert">
        <span v-if="isSaving" class="loading loading-spinner loading-sm"></span>
        {{ isEditing ? $t("notifications.alert-form.save") : $t("notifications.alert-form.create") }}
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { type LogEvent, type LogEntry, type LogMessage, asLogEntry } from "@/models/LogEntry";
import { Container } from "@/models/Container";
import type { ContainerJson } from "@/types/Container";
import { createExprEditor, createContainerHints, createLogHints } from "@/composable/exprEditor";

import type { Dispatcher, NotificationRule, PreviewResult } from "@/types/notifications";

const { close, onCreated, alert, prefill } = defineProps<{
  close?: () => void;
  onCreated?: () => void;
  alert?: NotificationRule;
  prefill?: { name?: string; containerExpression?: string; logExpression?: string };
}>();

// Fetch dispatchers
const destinations = ref<Dispatcher[]>([]);
onMounted(async () => {
  const res = await fetch(withBase("/api/notifications/dispatchers"));
  destinations.value = await res.json();
});

// Container store for autocomplete hints
const containerStore = useContainerStore();
const { containers } = storeToRefs(containerStore);
const containerNames = computed(() => [
  ...new Set(containers.value.filter((c) => c.state === "running").map((c) => c.name)),
]);
const imageNames = computed(() => [...new Set(containers.value.map((c) => c.image))]);
const hostNames = computed(() => [...new Set(containers.value.map((c) => c.host))]);

// Template refs
const alertNameInput = ref<HTMLInputElement>();
const containerEditorRef = ref<HTMLElement>();
const logEditorRef = ref<HTMLElement>();
const destinationDropdown = ref<HTMLDetailsElement>();

// Form state
const isEditing = computed(() => !!alert);
const alertName = ref(alert?.name ?? prefill?.name ?? "");
const containerExpression = ref(alert?.containerExpression ?? prefill?.containerExpression ?? "");
const logExpression = ref(alert?.logExpression ?? prefill?.logExpression ?? "");
const dispatcherId = ref(alert?.dispatcher?.id ?? 0);
const selectedDestination = computed(() => destinations.value.find((d) => d.id === dispatcherId.value));
useFocus(alertNameInput, { initialValue: true });

// Validation state
interface ContainerResult {
  error?: string;
  containers?: Container[];
}
const containerResult = ref<ContainerResult | null>(null);
const logError = ref<string | null>(null);
const logTotalCount = ref(0);
const logMessages = shallowRef<LogEntry<LogMessage>[]>([]);
const messageKeys = ref<string[]>([]);
const isLoading = ref(false);
const isSaving = ref(false);
const saveError = ref<string | null>(null);

const canSave = computed(
  () =>
    alertName.value.trim() &&
    containerExpression.value.trim() &&
    dispatcherId.value > 0 &&
    !containerResult.value?.error &&
    !logError.value &&
    !isSaving.value,
);

async function saveAlert() {
  if (!canSave.value) return;

  isSaving.value = true;
  saveError.value = null;

  try {
    const input = {
      name: alertName.value.trim(),
      containerExpression: containerExpression.value,
      logExpression: logExpression.value,
      dispatcherId: dispatcherId.value!,
      enabled: true,
    };

    const url = isEditing.value
      ? withBase(`/api/notifications/rules/${alert!.id}`)
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

    onCreated?.();
    close?.();
  } catch (e) {
    saveError.value = e instanceof Error ? e.message : "Failed to save alert";
  } finally {
    isSaving.value = false;
  }
}

async function validateExpressions() {
  if (!containerExpression.value && !logExpression.value) {
    containerResult.value = null;
    logError.value = null;
    logTotalCount.value = 0;
    logMessages.value = [];
    messageKeys.value = [];
    return;
  }

  isLoading.value = true;

  try {
    const res = await fetch(withBase("/api/notifications/preview"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        containerExpression: containerExpression.value,
        logExpression: logExpression.value || undefined,
      }),
    });

    if (!res.ok) {
      const errData = await res.json();
      throw new Error(errData.error || "Preview failed");
    }

    const data: PreviewResult = await res.json();

    // Update container result
    containerResult.value = containerExpression.value
      ? {
          error: data.containerError ?? undefined,
          containers: data.matchedContainers?.map((c) => Container.fromJSON(c as ContainerJson)),
        }
      : null;

    // Update message keys for autocomplete
    messageKeys.value = data.messageKeys ?? [];

    // Update log result
    if (logExpression.value && !data.containerError) {
      logError.value = data.logError ?? null;
      logTotalCount.value = data.totalLogs;
      logMessages.value = data.matchedLogs?.map((event) => asLogEntry(event as LogEvent)) ?? [];
    } else {
      logError.value = null;
      logTotalCount.value = 0;
      logMessages.value = [];
    }
  } catch (e) {
    containerResult.value = { error: e instanceof Error ? e.message : "Unknown error" };
  } finally {
    isLoading.value = false;
  }
}

const debouncedValidate = useDebounceFn(validateExpressions, 500);

watch(
  [containerExpression, logExpression],
  () => {
    isLoading.value = true;
    debouncedValidate();
  },
  { immediate: true },
);

let containerEditorView: Awaited<ReturnType<typeof createExprEditor>> | undefined;
let logEditorView: Awaited<ReturnType<typeof createExprEditor>> | undefined;

onMounted(async () => {
  if (containerEditorRef.value) {
    containerEditorView = await createExprEditor({
      parent: containerEditorRef.value,
      placeholder: 'name contains "api"',
      initialValue: alert?.containerExpression ?? prefill?.containerExpression ?? "",
      getHints: () => createContainerHints(containerNames.value, imageNames.value, hostNames.value),
      onChange: (v) => (containerExpression.value = v),
    });
  }

  if (logEditorRef.value) {
    logEditorView = await createExprEditor({
      parent: logEditorRef.value,
      placeholder: 'level == "error" && message contains "timeout"',
      initialValue: alert?.logExpression ?? prefill?.logExpression ?? "",
      getHints: () => createLogHints(messageKeys.value),
      onChange: (v) => (logExpression.value = v),
    });
  }
});

onScopeDispose(() => {
  containerEditorView?.destroy();
  logEditorView?.destroy();
});
</script>
