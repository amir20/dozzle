<template>
  <!-- Progress and completion are a thin sticky strip over the log stream; an empty
       result is not a status line but the whole answer, so it takes the room the log
       list would have filled. -->
  <div
    v-if="state === 'empty'"
    data-state="empty"
    class="flex flex-col items-center gap-3 px-4 py-12 text-center font-sans"
  >
    <div class="bg-base-content/5 text-base-content/40 flex size-10 items-center justify-center rounded-full">
      <mdi:text-search class="size-5" />
    </div>
    <div class="flex flex-col gap-1">
      <p class="text-sm font-semibold">{{ $t("label.search-status.empty") }}</p>
      <p class="text-base-content/50 text-xs">{{ $t("label.search-status.empty-hint") }}</p>
    </div>
  </div>
  <div
    v-else-if="state"
    :data-state="state"
    class="bg-base-200/80 text-base-content/70 sticky top-0 z-10 flex items-center gap-2 px-4 py-1.5 font-sans text-xs backdrop-blur"
  >
    <template v-if="state === 'searching'">
      <span>{{
        status.scannedTo ? $t("label.search-status.searching-to", { time }) : $t("label.search-status.searching")
      }}</span>
      <div class="ml-auto w-1/2">
        <IndeterminateBar color="primary" />
      </div>
    </template>
    <span v-else-if="state === 'capped'" class="tabular-nums">
      {{ $t("label.search-status.capped", { count: status.matches, time }) }}
    </span>
    <span v-else-if="state === 'exhausted'" class="tabular-nums">
      {{ $t("label.search-status.exhausted", { count: status.matches }) }}
    </span>
  </div>
</template>

<script lang="ts" setup>
import { type SearchStatus } from "@/composable/eventStreams";

const props = defineProps<{ status: SearchStatus }>();

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

const time = computed(() => (props.status.scannedTo ? new Date(props.status.scannedTo).toLocaleString() : ""));

const state = computed<"searching" | "empty" | "capped" | "exhausted" | null>(() => {
  if (showSearching.value) return "searching";
  if (!props.status.active && props.status.done) {
    if (props.status.matches === 0) return "empty";
    if (wasSlow.value) return props.status.reason === "capped" ? "capped" : "exhausted";
  }
  return null;
});
</script>
