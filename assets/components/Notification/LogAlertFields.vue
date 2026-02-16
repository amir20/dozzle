<template>
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

  <!-- Log Preview -->
  <div v-if="logMessages.length" class="mt-4">
    <div class="mb-2 text-lg">{{ $t("notifications.alert-form.preview") }}</div>
    <LogList
      :messages="logMessages"
      :last-selected-item="undefined"
      class="border-base-content/50 h-64 overflow-hidden rounded-lg border"
    />
  </div>
</template>

<script lang="ts" setup>
import { type LogEvent, type LogEntry, type LogMessage, asLogEntry } from "@/models/LogEntry";
import { createExprEditor, createLogHints } from "@/composable/exprEditor";
import type { NotificationRule, PreviewResult } from "@/types/notifications";

const props = defineProps<{
  alert?: NotificationRule;
  prefill?: { logExpression?: string };
  containerExpression: string;
  isLoading: boolean;
  validatePreview: (extra: Record<string, unknown>) => Promise<{ data: PreviewResult | null }>;
}>();

const logExpression = ref(props.alert?.logExpression ?? props.prefill?.logExpression ?? "");
const logError = ref<string | null>(null);
const logTotalCount = ref(0);
const logMessages = shallowRef<LogEntry<LogMessage>[]>([]);
const messageKeys = ref<string[]>([]);

const isLoading = computed(() => props.isLoading);

const canSave = computed(() => !logError.value);
const typeFields = computed(() => ({ logExpression: logExpression.value, metricExpression: "", cooldown: 0 }));

defineExpose({ canSave, typeFields });

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

// Editor
const logEditorRef = ref<HTMLElement>();
let logEditorView: Awaited<ReturnType<typeof createExprEditor>> | undefined;

onMounted(async () => {
  if (logEditorRef.value) {
    logEditorView = await createExprEditor({
      parent: logEditorRef.value,
      placeholder: 'level == "error" && message contains "timeout"',
      initialValue: props.alert?.logExpression ?? props.prefill?.logExpression ?? "",
      getHints: () => createLogHints(messageKeys.value),
      onChange: (v) => (logExpression.value = v),
    });
  }
});

onScopeDispose(() => {
  logEditorView?.destroy();
});
</script>
