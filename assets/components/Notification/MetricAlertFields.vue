<template>
  <fieldset class="fieldset">
    <legend class="fieldset-legend text-lg">Metric Expression</legend>
    <div
      class="input focus-within:input-primary w-full focus-within:z-50"
      :class="metricExpression.trim() && !metricError ? 'input-primary' : { 'input-error!': metricError }"
    >
      <div ref="metricEditorRef" class="w-full"></div>
    </div>
    <div v-if="metricError || metricExpression" class="fieldset-label">
      <span v-if="metricError" class="text-error">{{ metricError }}</span>
      <span v-else class="text-success">
        <mdi:check class="inline" />
        Expression is valid
      </span>
    </div>
    <p class="text-base-content/50 mt-1 text-xs">
      Available fields: <code>cpu</code> (CPU %), <code>memory</code> (memory %), <code>memoryUsage</code> (bytes)
    </p>
  </fieldset>

  <fieldset class="fieldset">
    <legend class="fieldset-legend text-lg">Cooldown (seconds)</legend>
    <input
      v-model.number="cooldown"
      type="number"
      min="10"
      class="input focus:input-primary w-full text-base"
      :class="cooldown > 0 ? 'input-primary' : ''"
      placeholder="300"
    />
    <p class="text-base-content/50 mt-1 text-xs">Minimum time between alerts per container. Default: 300s (5 min)</p>
  </fieldset>
</template>

<script lang="ts" setup>
import { createExprEditor, createMetricHints } from "@/composable/exprEditor";
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
const cooldown = ref(props.alert?.cooldown ?? 300);

const canSave = computed(() => !!metricExpression.value.trim() && !metricError.value);
const typeFields = computed(() => ({
  metricExpression: metricExpression.value,
  logExpression: "",
  cooldown: cooldown.value,
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
const metricEditorRef = ref<HTMLElement>();
let metricEditorView: Awaited<ReturnType<typeof createExprEditor>> | undefined;

onMounted(async () => {
  if (metricEditorRef.value) {
    metricEditorView = await createExprEditor({
      parent: metricEditorRef.value,
      placeholder: "cpu > 80 || memory > 90",
      initialValue: props.alert?.metricExpression ?? props.prefill?.metricExpression ?? "",
      getHints: () => createMetricHints(),
      onChange: (v) => (metricExpression.value = v),
    });
  }
});

onScopeDispose(() => {
  metricEditorView?.destroy();
});
</script>
