<template>
  <LogItem :logEntry>
    <div class="alert-row w-full border-l-3 pl-2" :data-level="level" :data-origin="alert.isOrigin">
      <!-- Follow-up anchor: the incident was already open and still firing
           here. One quiet line — a long incident must never draw two cards. -->
      <div v-if="!alert.isOrigin" class="flex items-center gap-2 text-xs opacity-60">
        <span class="dot"></span>
        <span>{{ $t("label.alert-still-firing") }} &mdash; {{ alert.headline }}</span>
        <a v-if="alert.url" :href="alert.url" target="_blank" rel="noopener" class="link">
          {{ $t("label.alert-open") }}
        </a>
      </div>

      <template v-else>
        <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
          <span class="chip">{{ $t("label.alert") }}</span>
          <span class="font-medium">{{ alert.headline }}</span>
          <!-- Only the count rides the one-line summary. Everything else —
               containers, what triage held back, the investigation — sits behind
               Details, so the row costs exactly one line until asked otherwise. -->
          <span class="text-xs opacity-60">{{ $t("label.alert-events", alert.eventCount) }}</span>

          <!-- Right-aligned so the headline starts at the same x-position on
               every alert down the stream, which is what makes a column of
               them scannable. -->
          <span class="ml-auto flex shrink-0 items-center gap-1">
            <button v-if="hasDetail" type="button" class="btn btn-ghost btn-xs" @click="expanded = !expanded">
              {{ $t("label.alert-details") }}
              <span class="caret" :data-open="expanded">&rsaquo;</span>
            </button>
            <a v-if="alert.url" :href="alert.url" target="_blank" rel="noopener" class="btn btn-xs">
              {{ $t("label.alert-view-in-cloud") }}
            </a>
          </span>
        </div>

        <div v-if="expanded" class="mt-1 flex flex-col gap-1 text-xs opacity-70">
          <div class="flex flex-wrap gap-x-4 gap-y-0.5">
            <span v-if="alert.containerCount && alert.containerCount > 1">
              {{ $t("label.alert-containers", alert.containerCount) }}
            </span>
            <span v-if="alert.suppressedCount">
              {{ $t("label.alert-held-back", alert.suppressedCount) }}
            </span>
          </div>
          <div v-if="alert.investigation" class="max-w-prose whitespace-pre-wrap">
            {{ alert.investigation }}
          </div>
        </div>
      </template>
    </div>
  </LogItem>
</template>

<script lang="ts" setup>
import { AlertLogEntry } from "@/models/LogEntry";

const { logEntry } = defineProps<{
  logEntry: AlertLogEntry;
  showContainerName?: boolean;
}>();

const alert = computed(() => logEntry.alert);
const expanded = ref(false);

/**
 * alerts.level is free text and defaults to '' — an alert that never went
 * through a summarizer has no level at all. A neutral fallback made the common
 * case indistinguishable from a log line, which is the one thing this row must
 * not be. Unknown reads as error.
 */
const level = computed(() => {
  const l = alert.value.level;
  if (l === "warn" || l === "warning") return "warn";
  if (l === "info" || l === "debug" || l === "trace") return "info";
  return "error";
});

const hasDetail = computed(
  () => !!alert.value.investigation || !!alert.value.suppressedCount || (alert.value.containerCount ?? 0) > 1,
);
</script>

<style scoped>
@reference "@/main.css";

/* No fill: a rail and a chip mark the row, and the log background is left
   alone. Colour is spent where the eye finds it fastest — a small,
   high-contrast marker against a calm surface. */
.alert-row {
  border-color: var(--tint);
}
.alert-row[data-level="error"] {
  --tint: var(--color-error);
  --tint-content: var(--color-error-content);
}
.alert-row[data-level="warn"] {
  --tint: var(--color-warning);
  --tint-content: var(--color-base-300);
}
.alert-row[data-level="info"] {
  --tint: var(--color-info);
  --tint-content: var(--color-info-content);
}

.chip {
  background-color: var(--tint);
  color: var(--tint-content);
  @apply shrink-0 rounded-xs px-1 text-[0.65rem] font-bold tracking-wider uppercase;
}

.dot {
  background-color: var(--tint);
  @apply size-1.5 shrink-0 rounded-full;
}

.caret {
  @apply inline-block transition-transform;
}
.caret[data-open="true"] {
  @apply rotate-90;
}
</style>
