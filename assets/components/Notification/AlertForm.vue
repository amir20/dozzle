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

    <!-- Alert Type Toggle -->
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">Alert Type</legend>
      <div class="flex gap-2">
        <button
          class="btn btn-sm"
          :class="alertType === 'log' ? 'btn-primary' : 'btn-outline'"
          @click="alertType = 'log'"
        >
          <mdi:text-box-outline class="mr-1" />
          Log Alert
        </button>
        <button
          class="btn btn-sm"
          :class="alertType === 'metric' ? 'btn-primary' : 'btn-outline'"
          @click="alertType = 'metric'"
        >
          <mdi:chart-line class="mr-1" />
          Metric Alert
        </button>
      </div>
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

    <!-- Type-specific fields -->
    <LogAlertFields
      v-if="alertType === 'log'"
      ref="fieldsRef"
      :alert="alert"
      :prefill="prefill"
      :container-expression="containerExpression"
      :is-loading="isLoading"
      :validate-preview="validatePreview"
    />
    <MetricAlertFields
      v-if="alertType === 'metric'"
      ref="fieldsRef"
      :alert="alert"
      :prefill="prefill"
      :container-expression="containerExpression"
      :is-loading="isLoading"
      :validate-preview="validatePreview"
    />

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

    <!-- Error -->
    <div v-if="saveError" class="alert alert-error">
      <span>{{ saveError }}</span>
    </div>

    <!-- Actions -->
    <div class="flex justify-end gap-2 pt-4">
      <button class="btn" @click="close?.()">{{ $t("notifications.alert-form.cancel") }}</button>
      <button class="btn btn-primary" :disabled="!canSave" @click="save">
        <span v-if="isSaving" class="loading loading-spinner loading-sm"></span>
        {{ isEditing ? $t("notifications.alert-form.save") : $t("notifications.alert-form.create") }}
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useAlertForm } from "@/composable/alertForm";
import LogAlertFields from "./LogAlertFields.vue";
import MetricAlertFields from "./MetricAlertFields.vue";
import type { NotificationRule } from "@/types/notifications";

const props = defineProps<{
  close?: () => void;
  onCreated?: () => void;
  alert?: NotificationRule;
  prefill?: { name?: string; containerExpression?: string; logExpression?: string; metricExpression?: string };
}>();

const {
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
} = useAlertForm(props);

// Template refs
const alertNameInput = ref<HTMLInputElement>();
const containerEditorRef = ref<HTMLElement>();
const destinationDropdown = ref<HTMLDetailsElement>();
const fieldsRef = ref<InstanceType<typeof LogAlertFields> | InstanceType<typeof MetricAlertFields>>();
useFocus(alertNameInput, { initialValue: true });

// Alert type
const alertType = ref<"log" | "metric">(props.alert?.metricExpression ? "metric" : "log");

const canSave = computed(() => baseCanSave.value && (fieldsRef.value?.canSave ?? false));

async function save() {
  if (!canSave.value || !fieldsRef.value) return;
  await saveAlert(fieldsRef.value.typeFields);
}

// Container editor
let containerEditorView: Awaited<ReturnType<typeof initContainerEditor>> | undefined;

onMounted(async () => {
  if (containerEditorRef.value) {
    containerEditorView = await initContainerEditor(containerEditorRef.value);
  }
});

onScopeDispose(() => {
  containerEditorView?.destroy();
});
</script>
