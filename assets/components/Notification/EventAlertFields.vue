<template>
  <ExpressionField
    v-model="eventExpression"
    :hint="$t('notifications.alert-form.event-filter-hint')"
    placeholder='name == "die"'
    :error="eventError"
    :examples="examples"
    :get-hints="getHints"
  />

  <DurationField
    v-model="cooldown"
    class="mt-4"
    :label="$t('notifications.alert-form.cooldown-label')"
    :hint="(duration: string) => $t('notifications.alert-form.cooldown-hint', { duration })"
    :zero-hint="$t('notifications.alert-form.no-cooldown')"
    :min="0"
    :max="3600"
    :step="10"
  />

  <!-- Preview: Docker events aren't retained, so show which representative events match -->
  <div class="mt-4">
    <div class="mb-2 flex items-center justify-between gap-2">
      <span class="text-base-content/70 text-sm font-semibold">{{ $t("notifications.alert-form.preview") }}</span>
      <span v-if="isLoading" class="loading loading-spinner loading-xs"></span>
      <span v-else-if="samples.length" class="text-sm" :class="matchCount ? 'text-success' : 'text-warning'">
        <mdi:check v-if="matchCount" class="inline" />
        <mdi:alert v-else class="inline" />
        {{
          matchCount
            ? $t("notifications.alert-form.event-matches", matchCount)
            : $t("notifications.alert-form.event-matches-none")
        }}
      </span>
    </div>

    <div
      v-if="samples.length"
      class="border-base-content/20 divide-base-content/10 min-h-24 divide-y rounded-lg border"
    >
      <div
        v-for="(sample, index) in samples"
        :key="index"
        class="flex items-center gap-2 px-3 py-1.5 text-sm"
        :class="sample.matches ? 'bg-success/10' : 'opacity-50'"
      >
        <mdi:check v-if="sample.matches" class="text-success shrink-0" />
        <mdi:minus v-else class="text-base-content/40 shrink-0" />
        <code class="font-mono">{{ sample.name }}</code>
        <span v-for="(value, key) in sample.attributes" :key="key" class="badge badge-sm badge-ghost font-mono">
          {{ key }}={{ value }}
        </span>
      </div>
    </div>
    <div
      v-else
      class="border-base-content/15 bg-base-content/[0.03] text-base-content/60 flex h-24 items-center justify-center rounded-lg border border-dashed px-4 text-center text-sm"
    >
      {{ emptyStateMessage }}
    </div>

    <p class="text-base-content/50 mt-1 min-h-4 text-xs">
      <template v-if="samples.length">{{ $t("notifications.alert-form.event-preview-note") }}</template>
    </p>
  </div>
</template>

<script lang="ts" setup>
import { createEventHints } from "@/composable/exprEditor";
import ExpressionField from "./ExpressionField.vue";
import DurationField from "./DurationField.vue";
import type { EventSample, NotificationRule, PreviewResult } from "@/types/notifications";

const props = defineProps<{
  alert?: NotificationRule;
  prefill?: { eventExpression?: string; cooldown?: number };
  containerExpression: string;
  hasContainers: boolean;
  isLoading: boolean;
  validatePreview: (extra: Record<string, unknown>) => Promise<{ data: PreviewResult | null }>;
}>();

const { t } = useI18n();

const eventExpression = ref(props.alert?.eventExpression ?? props.prefill?.eventExpression ?? "");
const eventError = ref<string | null>(null);
const cooldown = ref(props.alert?.cooldown ?? props.prefill?.cooldown ?? 10);
const samples = shallowRef<EventSample[]>([]);

const examples = [
  'name == "die"',
  'name in ["stop", "die"]',
  'name == "health_status" && attributes["healthStatus"] == "unhealthy"',
];
const getHints = () => createEventHints();

const canSave = computed(() => !!eventExpression.value.trim() && !eventError.value);
const typeFields = computed(() => ({
  eventExpression: eventExpression.value,
  logExpression: "",
  metricExpression: "",
  cooldown: cooldown.value,
  sampleWindow: 0,
}));

defineExpose({ canSave, typeFields });

const matchCount = computed(() => samples.value.filter((s) => s.matches).length);

const emptyStateMessage = computed(() => {
  if (!eventExpression.value.trim()) return t("notifications.alert-form.event-preview-hint");
  if (eventError.value) return t("notifications.alert-form.preview-fix-expression");
  return t("notifications.alert-form.event-preview-hint");
});

// Validation
async function validate() {
  if (!props.containerExpression && !eventExpression.value) {
    eventError.value = null;
    samples.value = [];
    return;
  }

  const { data } = await props.validatePreview({
    eventExpression: eventExpression.value || undefined,
  });

  if (data) {
    eventError.value = data.containerError ? null : (data.eventError ?? null);
    samples.value = data.eventError || data.containerError ? [] : (data.eventSamples ?? []);
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
</script>
