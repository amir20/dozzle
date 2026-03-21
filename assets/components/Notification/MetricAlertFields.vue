<template>
  <fieldset class="fieldset">
    <legend class="fieldset-legend text-lg">{{ $t("notifications.alert-form.metric-filter") }}</legend>
    <div
      class="input focus-within:input-primary w-full focus-within:z-50"
      :class="metricExpression.trim() && !metricError ? 'input-primary' : { 'input-error!': metricError }"
    >
      <div ref="editorRef" class="w-full"></div>
    </div>
    <div v-if="metricError || metricExpression" class="fieldset-label">
      <span v-if="metricError" class="text-error">{{ metricError }}</span>
      <span v-else class="text-success">
        <mdi:check class="inline" />
        {{ $t("notifications.alert-form.expression-valid") }}
      </span>
    </div>
    <p class="text-base-content/50 mt-1 text-xs">
      {{
        $t("notifications.alert-form.metric-fields-hint", {
          fields: "cpu (CPU %), memory (memory %), memoryUsage (bytes)",
        })
      }}
    </p>
  </fieldset>

  <fieldset class="fieldset">
    <legend class="fieldset-legend text-lg">{{ $t("notifications.alert-form.sample-window-label") }}</legend>
    <input v-model.number="sampleWindow" type="range" min="15" max="300" step="15" class="range range-primary" />
    <p class="text-base-content/50 mt-1 text-xs">
      {{
        $t("notifications.alert-form.sample-window-hint", {
          duration: formatDuration(sampleWindow, locale || undefined),
        })
      }}
    </p>
  </fieldset>

  <fieldset class="fieldset">
    <legend class="fieldset-legend text-lg">{{ $t("notifications.alert-form.cooldown-label") }}</legend>
    <input v-model.number="cooldown" type="range" min="10" max="3600" step="10" class="range range-primary" />
    <p class="text-base-content/50 mt-1 text-xs">
      {{ $t("notifications.alert-form.cooldown-hint", { duration: formatDuration(cooldown, locale || undefined) }) }}
    </p>
  </fieldset>
</template>

<script lang="ts" setup>
import { createMetricHints } from "@/composable/exprEditor";
import type { NotificationRule, PreviewResult } from "@/types/notifications";

const props = defineProps<{
  alert?: NotificationRule;
  prefill?: { metricExpression?: string };
  containerExpression: string;
  isLoading: boolean;
  validatePreview: (extra: Record<string, unknown>) => Promise<{ data: PreviewResult | null }>;
}>();

const metricExpression = ref(props.alert?.metricExpression ?? props.prefill?.metricExpression ?? "");
const metricError = ref<string | null>(null);
const sampleWindow = ref(props.alert?.sampleWindow ?? 15);
const cooldown = ref(props.alert?.cooldown ?? 300);

const canSave = computed(() => !!metricExpression.value.trim() && !metricError.value);
const typeFields = computed(() => ({
  metricExpression: metricExpression.value,
  logExpression: "",
  cooldown: cooldown.value,
  sampleWindow: sampleWindow.value,
}));

defineExpose({ canSave, typeFields });

// Validation
async function validate() {
  if (!props.containerExpression && !metricExpression.value) {
    metricError.value = null;
    return;
  }

  const { data } = await props.validatePreview({
    metricExpression: metricExpression.value || undefined,
  });

  if (data) {
    metricError.value = data.metricError ?? null;
  }
}

const debouncedValidate = useDebounceFn(validate, 500);
watch(
  [() => props.containerExpression, metricExpression],
  () => {
    debouncedValidate();
  },
  { immediate: true },
);

// Editor
const editorRef = ref<HTMLElement>();
useExprEditorField(editorRef, {
  placeholder: "cpu > 80 || memory > 90",
  initialValue: props.alert?.metricExpression ?? props.prefill?.metricExpression ?? "",
  getHints: () => createMetricHints(),
  onChange: (v) => (metricExpression.value = v),
});
</script>
