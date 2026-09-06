<template>
  <ExpressionField
    v-model="metricExpression"
    :hint="$t('notifications.alert-form.metric-filter-hint')"
    placeholder="cpu > 80 || memory > 90"
    :error="metricError"
    :examples="examples"
    :get-hints="getHints"
  />

  <DurationField
    v-model="sampleWindow"
    class="mt-4"
    :label="$t('notifications.alert-form.sample-window-label')"
    :hint="(duration: string) => $t('notifications.alert-form.sample-window-hint', { duration })"
    :min="15"
    :max="300"
    :step="15"
  />

  <DurationField
    v-model="cooldown"
    :label="$t('notifications.alert-form.cooldown-label')"
    :hint="(duration: string) => $t('notifications.alert-form.cooldown-hint', { duration })"
    :zero-hint="$t('notifications.alert-form.no-cooldown')"
    :min="0"
    :max="3600"
    :step="10"
  />

  <!-- Preview: the expression replayed over the stats Dozzle already has buffered -->
  <div class="mt-4">
    <div class="mb-2 flex items-center justify-between gap-2">
      <span class="text-base-content/70 text-sm font-semibold">{{ $t("notifications.alert-form.preview") }}</span>
      <span v-if="isLoading" class="loading loading-spinner loading-xs"></span>
      <span v-else-if="samples.length" class="text-sm" :class="triggerCount ? 'text-warning' : 'text-success'">
        <mdi:alert v-if="triggerCount" class="inline" />
        <mdi:check v-else class="inline" />
        {{
          triggerCount
            ? $t("notifications.alert-form.metric-would-fire", { count: triggerCount })
            : $t("notifications.alert-form.metric-would-not-fire")
        }}
      </span>
    </div>

    <div v-if="samples.length" class="border-base-content/20 overflow-x-auto rounded-lg border">
      <table class="table-sm table">
        <thead>
          <tr>
            <th>{{ $t("notifications.alert-form.metric-container") }}</th>
            <th class="text-right">{{ $t("notifications.alert-form.metric-cpu") }}</th>
            <th class="text-right">{{ $t("notifications.alert-form.metric-memory") }}</th>
            <th class="text-right">{{ $t("notifications.alert-form.metric-window-match") }}</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="sample in samples" :key="sample.containerId">
            <td class="max-w-40 truncate font-medium">{{ sample.name }}</td>
            <td class="text-right font-mono" :class="{ 'text-warning': sample.matches }">
              {{ sample.totalSamples ? `${sample.cpu.toFixed(1)}%` : "—" }}
            </td>
            <td class="text-right font-mono" :class="{ 'text-warning': sample.matches }">
              {{ sample.totalSamples ? `${sample.memory.toFixed(1)}%` : "—" }}
            </td>
            <td class="text-base-content/70 text-right font-mono">
              <template v-if="sample.totalSamples">{{ sample.matchedSamples }}/{{ sample.totalSamples }}</template>
              <template v-else>—</template>
            </td>
            <td class="text-right">
              <span v-if="sample.wouldTrigger" class="badge badge-sm badge-warning">
                {{ $t("notifications.alert-form.metric-firing") }}
              </span>
              <span v-else-if="!sample.totalSamples" class="text-base-content/50 text-xs">
                {{ $t("notifications.alert-form.metric-no-samples") }}
              </span>
              <span v-else-if="sample.matches" class="badge badge-sm badge-ghost">
                {{ $t("notifications.alert-form.metric-matching-now") }}
              </span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div
      v-else
      class="border-base-content/15 bg-base-content/[0.03] text-base-content/60 flex h-24 items-center justify-center rounded-lg border border-dashed px-4 text-center text-sm"
    >
      {{ emptyStateMessage }}
    </div>

    <p v-if="samples.length" class="text-base-content/50 mt-1 text-xs">
      {{ $t("notifications.alert-form.metric-preview-note", { duration: windowLabel }) }}
    </p>
  </div>
</template>

<script lang="ts" setup>
import { createMetricHints } from "@/composable/exprEditor";
import ExpressionField from "./ExpressionField.vue";
import DurationField from "./DurationField.vue";
import type { MetricSample, NotificationRule, PreviewResult } from "@/types/notifications";

const props = defineProps<{
  alert?: NotificationRule;
  prefill?: { metricExpression?: string; cooldown?: number; sampleWindow?: number };
  containerExpression: string;
  hasContainers: boolean;
  isLoading: boolean;
  validatePreview: (extra: Record<string, unknown>) => Promise<{ data: PreviewResult | null }>;
}>();

const { t } = useI18n();

const metricExpression = ref(props.alert?.metricExpression ?? props.prefill?.metricExpression ?? "");
const metricError = ref<string | null>(null);
const sampleWindow = ref(props.alert?.sampleWindow ?? props.prefill?.sampleWindow ?? 15);
const cooldown = ref(props.alert?.cooldown ?? props.prefill?.cooldown ?? 300);
const samples = shallowRef<MetricSample[]>([]);

const examples = ["cpu > 80", "memory > 90", "any(mounts, .usedPercent >= 85)"];
const getHints = () => createMetricHints();

const canSave = computed(() => !!metricExpression.value.trim() && !metricError.value);
const typeFields = computed(() => ({
  metricExpression: metricExpression.value,
  logExpression: "",
  cooldown: cooldown.value,
  sampleWindow: sampleWindow.value,
}));

defineExpose({ canSave, typeFields });

const triggerCount = computed(() => samples.value.filter((s) => s.wouldTrigger).length);
const windowLabel = computed(() => formatDuration(sampleWindow.value, locale.value || undefined));

const emptyStateMessage = computed(() => {
  if (!metricExpression.value.trim()) return t("notifications.alert-form.metric-preview-hint");
  if (!props.containerExpression.trim() || !props.hasContainers)
    return t("notifications.alert-form.preview-needs-containers");
  if (metricError.value) return t("notifications.alert-form.preview-fix-expression");
  return t("notifications.alert-form.metric-no-samples-hint");
});

// Validation
async function validate() {
  if (!props.containerExpression && !metricExpression.value) {
    metricError.value = null;
    samples.value = [];
    return;
  }

  const { data } = await props.validatePreview({
    metricExpression: metricExpression.value || undefined,
    sampleWindow: sampleWindow.value,
  });

  if (data) {
    metricError.value = data.containerError ? null : (data.metricError ?? null);
    samples.value = data.metricError || data.containerError ? [] : (data.metricSamples ?? []);
  }
}

const debouncedValidate = useDebounceFn(validate, 500);
watch(
  [() => props.containerExpression, metricExpression, sampleWindow],
  () => {
    debouncedValidate();
  },
  { immediate: true },
);
</script>
