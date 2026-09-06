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
  <div
    v-else
    :data-level="level"
    :data-position="position"
    class="mt-1.5 size-2.5 flex-none rounded-lg"
    :class="{ showUnknown }"
  ></div>
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
[data-position="start"],
[data-position="middle"],
[data-position="end"] {
  align-self: stretch;
  height: auto;
}

[data-position="start"] {
  border-radius: 0.375rem 0.375rem 0 0;
}

[data-position="middle"] {
  border-radius: 0;
  margin-top: 0;
}

[data-position="end"] {
  border-radius: 0 0 0.375rem 0.375rem;
  margin-top: 0;
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
