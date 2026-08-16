<template>
  <LogItem :logEntry>
    <div class="w-full" :data-origin="alert.isOrigin">
      <!-- Follow-up anchor: the incident was already open and still firing
           here. Deliberately a single quiet line — a full card at every
           follow-up would bury the logs the user came to read. -->
      <div v-if="!alert.isOrigin" class="text-base-content/60 flex items-center gap-2 text-xs">
        <span class="iconify lucide--activity size-3.5 shrink-0"></span>
        <span>{{ $t("label.alert-still-firing") }} &mdash; {{ alert.headline }}</span>
        <a v-if="alert.url" :href="alert.url" target="_blank" rel="noopener" class="link">{{
          $t("label.alert-open")
        }}</a>
      </div>

      <div v-else class="border-l-3 pl-3" :data-level="alert.level">
        <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
          <span class="iconify lucide--triangle-alert size-4 shrink-0"></span>
          <span class="font-semibold">{{ alert.headline }}</span>
          <a v-if="alert.url" :href="alert.url" target="_blank" rel="noopener" class="link text-xs">
            {{ $t("label.alert-view-in-cloud") }}
          </a>
        </div>

        <div class="text-base-content/60 mt-0.5 flex flex-wrap items-center gap-x-3 text-xs">
          <span>{{ $t("label.alert-events", alert.eventCount) }}</span>
          <span v-if="alert.containerCount && alert.containerCount > 1">
            {{ $t("label.alert-containers", alert.containerCount) }}
          </span>
          <span v-if="alert.suppressedCount">
            {{ $t("label.alert-held-back", alert.suppressedCount) }}
          </span>
        </div>

        <details v-if="alert.investigation" class="mt-1">
          <summary class="text-base-content/60 cursor-pointer text-xs select-none">
            {{ $t("label.alert-investigation") }}
          </summary>
          <div class="text-base-content/80 mt-1 max-w-prose text-xs whitespace-pre-wrap">
            {{ alert.investigation }}
          </div>
        </details>
      </div>
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
</script>

<style scoped>
@reference "@/main.css";
[data-level="error"],
[data-level="fatal"],
[data-level="critical"],
[data-level="severe"] {
  @apply border-error;
}
[data-level="warn"],
[data-level="warning"] {
  @apply border-warning;
}
[data-level="info"],
[data-level="debug"],
[data-level="trace"],
[data-level="unknown"] {
  @apply border-base-content/30;
}
</style>
