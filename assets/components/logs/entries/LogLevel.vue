<template>
  <!-- A matched notification takes over the level marker instead of adding a
       second icon beside it: the bell is tinted with the level colour, so one
       glyph still says both "this line fired" and what level it was. -->
  <span
    v-if="event"
    class="event-badge mt-1 flex w-2.5 flex-none justify-center"
    :data-event-level="level"
    :title="event.suppressed ? $t('label.event-suppressed-hint') : $t('label.event-sent-hint')"
  >
    <mdi:bell-off v-if="event.suppressed" class="size-3.5 shrink-0" />
    <mdi:bell-alert v-else class="size-3.5 shrink-0" />
  </span>
  <!-- A single line gets a dot; the rail is reserved for grouped entries, where
       its length is the thing it says (these lines are one entry). Both live in
       the same 10px slot, so the message column stays aligned whatever the
       marker is, and both are sized in em so they track the log font size. -->
  <div
    v-else
    class="level-rail flex w-2.5 flex-none justify-center"
    :class="position ? 'mt-1.5' : 'h-[1.55em] items-center'"
    :data-position="position"
  >
    <div
      :data-level="level"
      class="rounded-full"
      :class="[rail ? 'h-full w-[3px]' : 'size-[0.45em] min-h-1 min-w-1', { 'show-unknown': showUnknown }]"
    ></div>
  </div>
</template>
<script lang="ts" setup>
import { Position, Level, type MatchedEvent } from "@/models/LogEntry";

const {
  level,
  position,
  event,
  showUnknown = false,
} = defineProps<{
  level?: Level;
  position?: Position;
  event?: MatchedEvent;
  showUnknown?: boolean;
}>();

// A single error line gets the rail a grouped entry gets rather than a 4px dot:
// with the row background left neutral, this marker is the only thing carrying
// severity, so it has to be visible from a scroll.
const rail = computed(() => !!position || level === "error" || level === "fatal");
</script>

<style scoped>
@reference "@/main.css";
.level-rail[data-position="start"],
.level-rail[data-position="middle"],
.level-rail[data-position="end"] {
  align-self: stretch;
  height: auto;
}

.level-rail[data-position="middle"],
.level-rail[data-position="end"] {
  margin-top: 0;
}

/* Round only the outer ends so a grouped entry reads as one continuous rail. */
.level-rail[data-position="start"] > div {
  border-radius: 9999px 9999px 0 0;
}

.level-rail[data-position="middle"] > div {
  border-radius: 0;
}

.level-rail[data-position="end"] > div {
  border-radius: 0 0 9999px 9999px;
}

/* Named data-event-level, NOT data-level: the unscoped block below paints any
   element carrying data-level with an !important background. */
.event-badge {
  @apply text-base-content/60 transition-colors;
}
.event-badge[data-event-level="debug"],
.event-badge[data-event-level="trace"] {
  @apply text-purple;
}
.event-badge[data-event-level="info"] {
  @apply text-green;
}
.event-badge[data-event-level="error"],
.event-badge[data-event-level="fatal"] {
  @apply text-red;
}
.event-badge[data-event-level="warn"] {
  @apply text-orange;
}
</style>
<style>
@reference "@/main.css";
[data-level="debug"],
[data-level="trace"] {
  @apply !bg-purple;
}

[data-level="info"] {
  @apply !bg-green;
}

[data-level="error"],
[data-level="fatal"] {
  @apply !bg-red;
}

[data-level="warn"] {
  @apply !bg-orange;
}

[data-level="unknown"].show-unknown {
  @apply !bg-base-300;
}
</style>
