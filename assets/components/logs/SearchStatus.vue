<template>
  <!-- An empty result is not a status line but the whole answer, so it takes the
       room the log list would have filled. -->
  <EmptyState
    v-if="state === 'empty'"
    data-state="empty"
    :title="$t('label.search-status.empty')"
    :hint="$t('label.search-status.empty-hint')"
  >
    <template #icon><mdi:text-search class="size-6" /></template>
  </EmptyState>
  <!-- Same, while the search is still running with nothing yet to show: the page
       is otherwise blank, and a 12px line in the corner of an empty screen does
       not say "this is working on it". Once matches are on screen the list is
       the answer and progress goes back to being a thin strip over it. -->
  <EmptyState
    v-else-if="state === 'searching' && empty"
    data-state="searching"
    :title="$t('label.search-status.searching')"
    :hint="matches"
  >
    <template #icon><mdi:loading class="text-info size-6 motion-safe:animate-spin" /></template>
    <template #actions>
      <div class="flex flex-col items-center gap-2">
        <IndeterminateBar color="primary" class="w-40 overflow-hidden rounded-full" />
        <!-- The stamp walks backwards as the scan goes. It is the only readout
             that proves the search is moving and not wedged, so it keeps its
             seconds here even though the finished states drop them. -->
        <span v-if="scanned" class="text-base-content/40 font-mono text-xs tabular-nums" :title="exactTime">
          {{ scanned }}
        </span>
      </div>
    </template>
  </EmptyState>
  <!-- With matches already on screen this is a boundary marker, not a banner: it
       sits at the top of the list, which is the edge the search is working past
       and the place you end up when you scroll looking for older matches. It was
       sticky, which on the views where the document scrolls parked it under the
       title bar where nobody ever saw it. -->
  <div v-else-if="state" :data-state="state" class="border-base-content/10 bg-base-200/60 border-b font-sans">
    <!-- Everything stays left: the find box floats over the right end of this
         strip, and a detail parked under it is a detail nobody reads. -->
    <div class="flex items-center gap-2 px-4 py-2">
      <span class="flex size-5 shrink-0 items-center justify-center rounded-full" :class="tint">
        <mdi:loading v-if="state === 'searching'" class="size-3.5 motion-safe:animate-spin" />
        <mdi:check v-else-if="state === 'exhausted'" class="size-3.5" />
        <mdi:information-outline v-else class="size-3.5" />
      </span>
      <span class="text-base-content/70 truncate text-sm">{{ label }}</span>
      <span
        v-for="(d, i) in details"
        :key="d"
        class="text-base-content/40 flex shrink-0 items-center gap-2 font-mono text-xs tabular-nums"
        :title="exactTime"
      >
        <span v-if="i > 0" class="text-base-content/20" aria-hidden="true">·</span>
        {{ d }}
      </span>
    </div>
    <!-- The sweep is the strip's own bottom edge rather than a bar floating in
         the row, so the whole header reads as the thing that is working. -->
    <IndeterminateBar v-if="state === 'searching'" color="primary" class="-mb-px" />
  </div>
</template>

<script lang="ts" setup>
import { type SearchStatus } from "@/composable/logs/eventStreams";

const props = defineProps<{
  status: SearchStatus;
  /** The stream has nothing on screen yet, so progress is the whole page. */
  empty?: boolean;
}>();
const { t } = useI18n();

// Reveal the in-progress bar only after a short delay so fast searches (the
// common case, which return almost instantly) never flash it. Slow searches
// — sparse matches over a large log — are the only ones that surface it.
const showSearching = ref(false);
// Remember whether this search ever ran slow, so the completion summary only
// shows for searches that actually made the user wait.
const wasSlow = ref(false);

// Watch the boolean, not the whole status object: a slow search replaces the
// status object on every progress event, and re-arming the timer each time
// would keep the bar from ever appearing. The delay must measure from when the
// search started.
const active = computed(() => props.status.active);
let timer: ReturnType<typeof setTimeout> | undefined;
watch(
  active,
  (isActive) => {
    clearTimeout(timer);
    if (isActive) {
      timer = setTimeout(() => {
        showSearching.value = true;
        wasSlow.value = true;
      }, 400);
    } else {
      showSearching.value = false;
    }
  },
  { immediate: true },
);
onScopeDispose(() => clearTimeout(timer));

// An unparseable stamp drops the detail rather than formatting to junk: the
// headline is the part that matters and it stands on its own.
const scannedTo = computed(() => {
  if (!props.status.scannedTo) return undefined;
  const date = new Date(props.status.scannedTo);
  return Number.isNaN(date.getTime()) ? undefined : date;
});
const exactTime = computed(() => scannedTo.value?.toLocaleString());

const state = computed<"searching" | "empty" | "capped" | "exhausted" | null>(() => {
  if (showSearching.value) return "searching";
  if (!props.status.active && props.status.done) {
    if (props.status.matches === 0) return "empty";
    if (wasSlow.value) return props.status.reason === "capped" ? "capped" : "exhausted";
  }
  return null;
});

// State rides on the glyph so the strip itself stays neutral against the log
// text it sits on.
const tint = computed(() => (state.value === "exhausted" ? "bg-success/10 text-success" : "bg-info/10 text-info"));

// The headline says what happened; the count or the scan boundary trails it as
// a quiet mono detail rather than being welded into the same sentence.
const label = computed(() => {
  switch (state.value) {
    case "exhausted":
      return t("label.search-status.searched-all");
    case "capped":
      return t("label.search-status.matches", { count: props.status.matches });
    default:
      return t("label.search-status.searching");
  }
});

// How far back the scan reached is a boundary, not staleness, so it is an
// absolute stamp trimmed to what tells the two ends of a log apart. The full one
// is a hover away. Seconds survive the trim only while the search is running,
// where this stamp walking backwards is the proof that it is still moving.
const scanned = computed(() => {
  if (!scannedTo.value) return "";
  const time = new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    ...(state.value === "searching" ? { second: "2-digit" } : {}),
  }).format(scannedTo.value);
  return t("label.search-status.scanned-to", { time });
});

// The running total, which climbs while the search walks backwards. Nothing
// showed it before the search finished, which left the in-progress state saying
// only that something was happening and not how well it was going.
const matches = computed(() =>
  props.status.matches > 0 ? t("label.search-status.matches", { count: props.status.matches }) : "",
);

// The headline says what happened; the count and the scan boundary trail it as
// quiet mono details rather than being welded into the same sentence.
const details = computed(() => {
  switch (state.value) {
    case "exhausted":
      return [t("label.search-status.matches", { count: props.status.matches })];
    case "capped":
      return [scanned.value].filter(Boolean);
    default:
      return [matches.value, scanned.value].filter(Boolean);
  }
});
</script>
