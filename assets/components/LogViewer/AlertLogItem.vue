<template>
  <LogItem :logEntry>
    <div class="alert-band w-full rounded-xs border-l-4 px-3 py-1.5" :data-level="level" :data-origin="alert.isOrigin">
      <!-- Follow-up anchor: the incident was already open and still firing
           here. Deliberately one quiet line — a full card at every follow-up
           would bury the logs the user came to read. -->
      <template v-if="!alert.isOrigin">
        <div class="flex items-center gap-2 text-xs">
          <mdi:repeat class="accent size-3.5 shrink-0" />
          <span class="accent font-semibold uppercase">{{ $t("label.alert-still-firing") }}</span>
          <span class="opacity-70">{{ alert.headline }}</span>
          <a v-if="alert.url" :href="alert.url" target="_blank" rel="noopener" class="link">
            {{ $t("label.alert-open") }}
          </a>
        </div>
      </template>

      <template v-else>
        <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
          <mdi:alert class="accent size-4 shrink-0" />
          <span class="accent text-xs font-bold tracking-wide uppercase">{{ $t("label.alert") }}</span>
          <span class="font-semibold">{{ alert.headline }}</span>
          <a v-if="alert.url" :href="alert.url" target="_blank" rel="noopener" class="link text-xs">
            {{ $t("label.alert-view-in-cloud") }}
          </a>
        </div>

        <div class="mt-0.5 flex flex-wrap items-center gap-x-3 text-xs opacity-70">
          <span>{{ $t("label.alert-events", alert.eventCount) }}</span>
          <span v-if="alert.containerCount && alert.containerCount > 1">
            {{ $t("label.alert-containers", alert.containerCount) }}
          </span>
          <span v-if="alert.suppressedCount">
            {{ $t("label.alert-held-back", alert.suppressedCount) }}
          </span>
        </div>

        <details v-if="alert.investigation" class="mt-1">
          <summary class="cursor-pointer text-xs opacity-70 select-none">
            {{ $t("label.alert-investigation") }}
          </summary>
          <div class="mt-1 max-w-prose text-xs whitespace-pre-wrap opacity-80">
            {{ alert.investigation }}
          </div>
        </details>
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

/**
 * alerts.level is free text and defaults to '' — an alert that never went
 * through a summarizer has no level at all. Falling back to a neutral grey
 * made the common case indistinguishable from a log line, which is the one
 * thing this row must not be. Unknown reads as error.
 */
const level = computed(() => {
  const l = alert.value.level;
  if (l === "warn" || l === "warning") return "warn";
  if (l === "info" || l === "debug" || l === "trace") return "info";
  return "error";
});
</script>

<style scoped>
/* Plain CSS rather than @apply: the opacity modifier in `@apply bg-error/10`
   silently produced a SOLID fill, painting the row as a block of colour with
   its text invisible on top. color-mix is what main.css already uses for
   tinted surfaces, and it composes predictably here.

   The band is tinted; the text is not. Only the icon and the ALERT label take
   the level colour — colouring the headline too put red text on a red
   background. */
.alert-band {
  border-color: var(--tint);
  background-color: color-mix(in oklab, var(--tint) 14%, transparent);
}

.alert-band .accent {
  color: var(--tint);
}

.alert-band[data-level="error"] {
  --tint: var(--color-error);
}
.alert-band[data-level="warn"] {
  --tint: var(--color-warning);
}
.alert-band[data-level="info"] {
  --tint: var(--color-info);
}
</style>
