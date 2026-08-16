<template>
  <LogItem :logEntry>
    <div
      class="alert-row w-full min-w-0 border-l-3 pl-4 font-sans"
      :data-origin="alert.isOrigin"
      :data-alert-level="level"
    >
      <!-- Follow-up anchor: the incident was already open and still firing
           here. One quiet line — a long incident must never draw two cards. -->
      <div v-if="!alert.isOrigin" class="flex flex-wrap items-center gap-x-2 gap-y-1 py-0.5">
        <span class="chip chip-quiet">
          <mdi:bell-off class="size-3" />
          {{ $t("label.alert-suppressed") }}
        </span>
        <span class="opacity-70">{{ alert.headline }}</span>
        <span class="shrink-0 opacity-40">&middot;</span>
        <span class="shrink-0 text-xs opacity-50">{{ $t("label.alert-still-firing") }}</span>
        <a v-if="alert.url" :href="alert.url" target="_blank" rel="noopener" class="act ml-auto">
          {{ $t("label.alert-open") }}
        </a>
      </div>

      <template v-else>
        <div class="flex flex-col gap-1.5 py-1">
          <div class="flex flex-wrap items-center gap-x-2.5 gap-y-1">
            <span class="chip">
              <mdi:bell-alert class="size-3" />
              {{ $t("label.alert") }}
            </span>
            <span class="font-semibold">{{ alert.headline }}</span>
            <!-- Only the count rides the summary line. Everything else —
                 containers, what triage held back, the investigation — sits
                 behind Details. -->
            <span class="shrink-0 opacity-40">&middot;</span>
            <span class="shrink-0 opacity-60">
              {{ $t("label.alert-events", alert.eventCount) }}<template v-if="ranFor">, {{ ranFor }}</template>
            </span>
          </div>

          <!-- Actions get their own line rather than riding the headline: at
               the end of a long headline they were easy to miss, and a fixed
               position means they land in the same place on every alert down
               the stream. -->
          <div class="flex items-center justify-end gap-1.5">
            <button v-if="hasDetail" type="button" class="act" @click="expanded = !expanded">
              {{ $t("label.alert-details") }}
              <span class="caret" :data-open="expanded">&rsaquo;</span>
            </button>
            <a v-if="alert.url" :href="alert.url" target="_blank" rel="noopener" class="act act-tinted">
              {{ $t("label.alert-view-in-cloud") }}
            </a>
          </div>
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
          <!-- summary is the alert's description and is usually all a
               non-triaged alert has; investigation is triage's longer write-up
               and exists only when triage actually ran. -->
          <div v-if="alert.summary" class="max-w-prose whitespace-pre-wrap">{{ alert.summary }}</div>
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

// Anything under a minute is one moment, not a duration — Cloud's aggregation
// window is 15s, so sub-minute spans are an artefact of batching rather than a
// real incident length.
const MIN_REPORTABLE_MS = 60_000;

/**
 * How long the incident kept firing. Cloud folds follow-up batches into the
 * original alert, so lastActivityAt is already on the row — without it a card
 * reads as a single moment when the incident may have run for hours.
 */
const ranFor = computed(() => {
  const { lastActivityAt, ts } = alert.value;
  if (!lastActivityAt) return null;
  const ms = (lastActivityAt - ts) / 1_000_000;
  if (ms < MIN_REPORTABLE_MS) return null;
  return formatDuration(Math.round(ms / 1000), locale.value === "" ? undefined : locale.value);
});

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
  () =>
    !!alert.value.summary ||
    !!alert.value.investigation ||
    !!alert.value.suppressedCount ||
    (alert.value.containerCount ?? 0) > 1,
);
</script>

<style scoped>
@reference "@/main.css";

/* font-sans, not the monospace the log list sets on its <ul>: this row is
   prose about the logs rather than log output, and monospace made a sentence
   of it read as another line of the stream.

   No fill: a rail and a chip mark the row, and the log background is left
   alone. Colour is spent where the eye finds it fastest — a small,
   high-contrast marker against a calm surface.

   Keyed on data-alert-level, NOT data-level: LogLevel.vue ships a second,
   UNSCOPED style block whose `[data-level="error"] { @apply !bg-red }` paints
   any element in the app carrying that attribute, !important and all. Reusing
   the name filled this row solid red and made every local rule unwinnable. */
.alert-row {
  border-color: var(--tint);
}
.alert-row[data-alert-level="error"] {
  --tint: var(--color-error);
  --tint-content: var(--color-error-content);
}
.alert-row[data-alert-level="warn"] {
  --tint: var(--color-warning);
  --tint-content: var(--color-base-300);
}
.alert-row[data-alert-level="info"] {
  --tint: var(--color-info);
  --tint-content: var(--color-info-content);
}

.chip {
  background-color: var(--tint);
  color: var(--tint-content);
  @apply inline-flex shrink-0 items-center gap-1 rounded-xs px-1.5 py-px text-[0.62rem] font-bold tracking-wider uppercase;
}

.chip-quiet {
  background-color: transparent;
  color: var(--tint);
  border: 1px solid color-mix(in oklab, var(--tint) 45%, transparent);
}

/* Actions read as controls, not as more log text: a hairline button for the
   secondary one and a tint of the level colour for the primary. Deliberately
   not daisyUI's .btn — btn-ghost renders borderless (invisible as a control)
   and plain .btn renders a heavy filled box that outweighs the row. */
.act {
  @apply inline-flex shrink-0 items-center gap-1 rounded border px-2 py-0.5 text-[0.7rem] leading-normal;
  border-color: color-mix(in oklab, var(--color-base-content) 22%, transparent);
}
.act:hover {
  background-color: color-mix(in oklab, var(--color-base-content) 10%, transparent);
}
.act:focus-visible {
  @apply outline-primary outline-2 outline-offset-1;
}
.act-tinted {
  border-color: transparent;
  background-color: color-mix(in oklab, var(--tint) 20%, transparent);
  color: var(--tint);
  @apply font-semibold;
}
.act-tinted:hover {
  background-color: color-mix(in oklab, var(--tint) 30%, transparent);
}

.caret {
  @apply inline-block transition-transform;
}
.caret[data-open="true"] {
  @apply rotate-90;
}
</style>
