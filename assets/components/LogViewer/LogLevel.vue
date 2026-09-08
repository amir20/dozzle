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
  <!-- The rail is 3px inside a 10px slot: it stays a quiet colour cue at the
       edge of the text instead of a bullet competing with the first word, and
       the slot keeps every message column aligned whatever the marker is. -->
  <div v-else class="level-rail mt-1.5 flex w-2.5 flex-none justify-center" :data-position="position">
    <div
      :data-level="level"
      class="w-[3px] rounded-full"
      :class="[position ? 'h-full' : 'h-[0.9em] min-h-3', { 'show-unknown': showUnknown }]"
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
