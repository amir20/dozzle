<template>
  <ExpressionField
    v-model="logExpression"
    :hint="$t('notifications.alert-form.log-filter-hint')"
    placeholder='level == "error" && message contains "timeout"'
    :error="logError"
    :examples="examples"
    :get-hints="getHints"
  />

  <!-- Preview -->
  <div class="mt-4">
    <div class="mb-2 flex items-center justify-between gap-2">
      <span class="text-base-content/70 text-sm font-semibold">{{ $t("notifications.alert-form.preview") }}</span>
      <span v-if="isLoading" class="loading loading-spinner loading-xs"></span>
      <span v-else-if="!logError && logExpression.trim() && hasContainers" class="text-sm">
        <span v-if="logMessages.length" class="text-success">
          <mdi:check class="inline" />
          {{ $t("notifications.alert-form.logs-match", { count: logTotalCount, window: windowLabel }, logTotalCount) }}
        </span>
        <span v-else class="text-warning">
          <mdi:alert class="inline" />
          {{ $t("notifications.alert-form.no-logs-match", { window: windowLabel }) }}
        </span>
      </span>
    </div>

    <LogList
      v-if="logMessages.length"
      :messages="logMessages"
      :last-selected-item="undefined"
      class="border-base-content/20 bg-base-200/40 h-64 overflow-hidden rounded-lg border"
    />
    <!-- Same height as the log list so results appearing and disappearing don't resize the form. -->
    <div
      v-else
      class="border-base-content/15 bg-base-content/[0.03] text-base-content/60 flex h-64 items-center justify-center rounded-lg border border-dashed px-4 text-center text-sm"
    >
      {{ emptyStateMessage }}
    </div>

    <p class="text-base-content/50 mt-1 min-h-4 text-xs">
      <template v-if="truncated">{{ $t("notifications.alert-form.log-scan-note", scannedContainers) }}</template>
    </p>
  </div>
</template>

<script lang="ts" setup>
import { type LogEvent, type LogEntry, type LogMessage, asLogEntry } from "@/models/LogEntry";
import { createLogHints } from "@/composable/exprEditor";
import ExpressionField from "./ExpressionField.vue";
import type { NotificationRule, PreviewResult } from "@/types/notifications";

const props = defineProps<{
  alert?: NotificationRule;
  prefill?: { logExpression?: string };
  containerExpression: string;
  hasContainers: boolean;
  isLoading: boolean;
  validatePreview: (extra: Record<string, unknown>) => Promise<{ data: PreviewResult | null }>;
}>();

const { t } = useI18n();

const logExpression = ref(props.alert?.logExpression ?? props.prefill?.logExpression ?? "");
const logError = ref<string | null>(null);
const logTotalCount = ref(0);
const logMessages = shallowRef<LogEntry<LogMessage>[]>([]);
const messageKeys = ref<string[]>([]);
const logWindowSeconds = ref(7200);
const scannedContainers = ref(0);
const matchedContainerCount = ref(0);

const examples = ['level == "error"', 'message contains "timeout"', 'stream == "stderr"'];
const getHints = () => createLogHints(messageKeys.value);

// An empty expression is not "match everything" — the backend treats a rule without a log
// expression as inert — so it is a required field, not an optional filter.
const canSave = computed(() => !!logExpression.value.trim() && !logError.value);
const typeFields = computed(() => ({ logExpression: logExpression.value, metricExpression: "", cooldown: 0 }));

defineExpose({ canSave, typeFields });

const windowLabel = computed(() => formatDuration(logWindowSeconds.value, locale.value || undefined));
const truncated = computed(() => scannedContainers.value > 0 && matchedContainerCount.value > scannedContainers.value);

const emptyStateMessage = computed(() => {
  if (!logExpression.value.trim()) return t("notifications.alert-form.log-preview-hint");
  if (!props.containerExpression.trim() || !props.hasContainers)
    return t("notifications.alert-form.preview-needs-containers");
  if (logError.value) return t("notifications.alert-form.preview-fix-expression");
  return t("notifications.alert-form.no-logs-match", { window: windowLabel.value });
});

// Validation
async function validate() {
  if (!props.containerExpression && !logExpression.value) {
    logError.value = null;
    logTotalCount.value = 0;
    logMessages.value = [];
    messageKeys.value = [];
    return;
  }

  const { data } = await props.validatePreview({
    logExpression: logExpression.value || undefined,
  });

  if (data) {
    messageKeys.value = data.messageKeys ?? [];
    logWindowSeconds.value = data.logWindowSeconds || logWindowSeconds.value;
    scannedContainers.value = data.scannedContainers ?? 0;
    matchedContainerCount.value = data.matchedContainers?.length ?? 0;
    if (logExpression.value && !data.containerError) {
      logError.value = data.logError ?? null;
      logTotalCount.value = data.totalLogs;
      logMessages.value = data.matchedLogs?.map((event) => asLogEntry(event as LogEvent)) ?? [];
    } else {
      logError.value = null;
      logTotalCount.value = 0;
      logMessages.value = [];
    }
  }
}

const debouncedValidate = useDebounceFn(validate, 500);
watch(
  [() => props.containerExpression, logExpression],
  () => {
    debouncedValidate();
  },
  { immediate: true },
);
</script>
