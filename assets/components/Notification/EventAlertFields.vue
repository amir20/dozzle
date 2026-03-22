<template>
  <fieldset class="fieldset">
    <legend class="fieldset-legend text-lg">{{ $t("notifications.alert-form.event-filter") }}</legend>
    <div
      class="input focus-within:input-primary w-full focus-within:z-50"
      :class="eventExpression.trim() && !eventError ? 'input-primary' : { 'input-error!': eventError }"
    >
      <div ref="editorRef" class="w-full"></div>
    </div>
    <div v-if="eventError || eventExpression" class="fieldset-label">
      <span v-if="eventError" class="text-error">{{ eventError }}</span>
      <span v-else class="text-success">
        <mdi:check class="inline" />
        {{ $t("notifications.alert-form.expression-valid") }}
      </span>
    </div>
    <p class="text-base-content/50 mt-1 text-xs">
      {{
        $t("notifications.alert-form.event-fields-hint", {
          fields: "name (start, stop, die, restart, health_status), attributes (exitCode, signal, etc.)",
        })
      }}
    </p>
  </fieldset>

  <fieldset class="fieldset">
    <legend class="fieldset-legend text-lg">{{ $t("notifications.alert-form.cooldown-label") }}</legend>
    <input v-model.number="cooldown" type="range" min="0" max="3600" step="10" class="range range-primary" />
    <p class="text-base-content/50 mt-1 text-xs">
      {{ $t("notifications.alert-form.cooldown-hint", { duration: formatDuration(cooldown, locale || undefined) }) }}
    </p>
  </fieldset>
</template>

<script lang="ts" setup>
import { createEventHints } from "@/composable/exprEditor";
import type { NotificationRule, PreviewResult } from "@/types/notifications";

const props = defineProps<{
  alert?: NotificationRule;
  prefill?: { eventExpression?: string };
  containerExpression: string;
  isLoading: boolean;
  validatePreview: (extra: Record<string, unknown>) => Promise<{ data: PreviewResult | null }>;
}>();

const eventExpression = ref(props.alert?.eventExpression ?? props.prefill?.eventExpression ?? "");
const eventError = ref<string | null>(null);
const cooldown = ref(props.alert?.cooldown ?? 10);

const canSave = computed(() => !!eventExpression.value.trim() && !eventError.value);
const typeFields = computed(() => ({
  eventExpression: eventExpression.value,
  logExpression: "",
  metricExpression: "",
  cooldown: cooldown.value,
  sampleWindow: 0,
}));

defineExpose({ canSave, typeFields });

// Validation
async function validate() {
  if (!props.containerExpression && !eventExpression.value) {
    eventError.value = null;
    return;
  }

  const { data } = await props.validatePreview({
    eventExpression: eventExpression.value || undefined,
  });

  if (data) {
    eventError.value = data.eventError ?? null;
  }
}

const debouncedValidate = useDebounceFn(validate, 500);
watch(
  [() => props.containerExpression, eventExpression],
  () => {
    debouncedValidate();
  },
  { immediate: true },
);

// Editor
const editorRef = ref<HTMLElement>();
useExprEditorField(editorRef, {
  placeholder: 'name == "die"',
  initialValue: props.alert?.eventExpression ?? props.prefill?.eventExpression ?? "",
  getHints: () => createEventHints(),
  onChange: (v) => (eventExpression.value = v),
});
</script>
