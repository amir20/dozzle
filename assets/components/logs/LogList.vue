<template>
  <ul
    ref="list"
    class="group pt-4"
    :class="{ 'disable-wrap': !softWrap, [size]: true, compact, 'highlight-errors': highlightErrors }"
    data-logs
  >
    <li
      v-for="item in messages"
      :key="item.id"
      :id="item.id.toString()"
      :data-time="item.date.getTime()"
      :data-log-level="rowLevel(item)"
      class="group/entry"
      :class="{ 'log-permalink-target': permalinkLogId === item.id.toString() }"
    >
      <component :is="item.getComponent()" :log-entry="item" />
    </li>
  </ul>
</template>

<script lang="ts" setup>
import { AlertLogEntry, CloudEventLogEntry, type LogEntry, type LogMessage } from "@/models/LogEntry";

const { progress, currentDate, available, paused } = useScrollContext();

const { messages } = defineProps<{
  messages: LogEntry<LogMessage>[];
}>();

const { containers } = useLoggingContext();

// Only real log output gets the row tint. Alert and cloud-event rows already
// carry their own level marker and deliberately leave the row background alone,
// so tinting them would fight styling they own.
const rowLevel = (item: LogEntry<LogMessage>) =>
  item instanceof AlertLogEntry || item instanceof CloudEventLogEntry ? undefined : item.level;

const route = useRoute();
const permalinkLogId = computed(() => (typeof route.query.logId === "string" ? route.query.logId : ""));

const list = ref<HTMLElement>();

// Only a single container has a lifetime to place a log on; merged and grouped
// views have nothing to measure against, so they say so instead of leaving the
// readout parked at the default 100%.
watchEffect(() => (available.value = containers.value.length === 1));

// Docker leaves finishedAt at the zero time for a container that never stopped.
const isSet = (date: Date) => date.getFullYear() > 1;

// The position is read off one fixed line, the middle of the visible column,
// rather than from whichever rows last crossed an observer's edge. That makes
// it a pure function of the scroll offset: the same spot always gives the same
// value whichever way you scrolled to it, tall rows are measured like any
// other, and nothing resets when rows are appended or older ones load in.
// Blending between the row under the line and the next one keeps it moving
// continuously instead of stepping a row at a time.
function measure() {
  const ul = list.value;
  if (!ul || !paused.value || containers.value.length !== 1) return;
  const rows = ul.children;
  if (rows.length === 0) return;

  const scroller = ul.closest<HTMLElement>("[data-scrolling]")?.getBoundingClientRect();
  const top = Math.max(scroller?.top ?? 0, 0);
  const bottom = Math.min(scroller?.bottom ?? window.innerHeight, window.innerHeight);
  const line = (top + bottom) / 2;

  // Rows are in time order, so the last one starting above the line is found
  // by bisection: a handful of layout reads per frame however long the list.
  let lo = 0;
  let hi = rows.length - 1;
  while (lo < hi) {
    const mid = (lo + hi + 1) >> 1;
    if (rows[mid].getBoundingClientRect().top <= line) lo = mid;
    else hi = mid - 1;
  }

  const row = rows[lo];
  const rect = row.getBoundingClientRect();
  const from = Number(row.getAttribute("data-time"));
  const next = rows[lo + 1]?.getAttribute("data-time");
  if (!Number.isFinite(from)) return;
  const through = rect.height > 0 ? Math.min(Math.max((line - rect.top) / rect.height, 0), 1) : 0;
  const time = next ? from + (Number(next) - from) * through : from;

  const container = containers.value[0];
  const running = container.state === "running" || container.state === "paused";
  const end = !running && isSet(container.finishedAt) ? container.finishedAt.getTime() : Date.now();
  const span = end - container.created.getTime();
  progress.value = span > 0 ? (time - container.created.getTime()) / span : 1;
  currentDate.value = new Date(time);
}

let frame = 0;
const schedule = () => {
  if (frame) return;
  frame = requestAnimationFrame(() => {
    frame = 0;
    measure();
  });
};
onScopeDispose(() => cancelAnimationFrame(frame));

// Scroll does not bubble, so listening in the capture phase on window hears
// whichever element is scrolling: the page, or this column's own scroller.
useEventListener(window, "scroll", schedule, { capture: true, passive: true });
useResizeObserver(list, schedule);
useMutationObserver(list, schedule, { childList: true });
watch(paused, schedule);
</script>
<style scoped>
@reference "@/main.css";
ul {
  font-family: var(--font-mono);
  line-height: 1.55;

  > li {
    /* pl leaves an empty gutter on desktop so the hover actions button has a
       home of its own instead of covering the timestamp. */
    @apply flex px-2 py-1 break-words transition-colors duration-75 last:snap-end md:pr-4 md:pl-9;
    &:last-child {
      scroll-margin-block-end: 5rem;
    }

    /* Written long-hand rather than as odd:/hover: utilities because the order
       below is the whole point: hover has to beat the zebra, and the error tint
       has to beat both. */
    &:nth-child(odd) {
      background-color: color-mix(in oklab, var(--color-base-content) 2.5%, transparent);
    }

    &:hover {
      background-color: color-mix(in oklab, var(--color-base-content) 8%, transparent);
    }

    &.log-permalink-target {
      @apply bg-secondary/15;
      animation: log-permalink-pulse 1.4s ease-out;
    }
  }

  /* Severity primarily rides on the level rail in LogLevel.vue, so this is a
     hint rather than the signal, and warn does not get one at all: an orange
     wash on a routine retry line was the noisiest thing in the stream. Off by
     choice for anyone who wants the field completely flat. */
  &.highlight-errors > li {
    &[data-log-level="error"],
    &[data-log-level="fatal"] {
      background-color: color-mix(in oklab, var(--color-red) 5%, transparent);
    }

    &[data-log-level="error"]:hover,
    &[data-log-level="fatal"]:hover {
      background-color: color-mix(in oklab, var(--color-red) 12%, transparent);
    }
  }

  &.small {
    @apply text-[0.7em];
  }

  &.medium {
    @apply text-[0.8em];
  }

  &.large {
    @apply text-[1em];
  }

  &.compact {
    > li {
      @apply py-0;
    }

    :deep(.tag) {
      @apply rounded-none;
    }
  }

  /* A soft-wrapped line hangs its continuation past the start of the entry, so
     one long line can never be mistaken for two. Harmless when wrapping is off:
     the negative indent and the padding cancel out. */
  :deep(.log-message) {
    padding-left: 1.5ch;
    text-indent: -1.5ch;
  }

  :deep(mark) {
    @apply bg-secondary inline-block rounded-xs;
    animation: pops 200ms ease-out;
  }

  :deep(a[rel~="external"]) {
    @apply text-primary underline-offset-4 hover:underline;
  }
}

@keyframes pops {
  0% {
    transform: scale(1.5);
  }
  100% {
    transform: scale(1.05);
  }
}

@keyframes log-permalink-pulse {
  0% {
    background-color: var(--color-secondary);
  }
  100% {
    /* Settle to the resting bg-secondary/15 declared on the .li above. */
    background-color: color-mix(in oklab, var(--color-secondary) 15%, transparent);
  }
}
</style>
