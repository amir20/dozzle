<template>
  <ul class="group pt-4" :class="{ 'disable-wrap': !softWrap, [size]: true, compact }" data-logs>
    <li
      v-for="item in messages"
      ref="list"
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

const { progress, currentDate, available } = useScrollContext();

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

const list = ref<HTMLElement[]>([]);

let previousDate = new Date();
// Only a single container has a lifetime to place a log on; merged and grouped
// views have nothing to measure against, so they say so instead of leaving the
// readout parked at the default 100%.
watchEffect(() => (available.value = containers.value.length === 1));
useIntersectionObserver(
  list,
  (entries) => {
    if (containers.value.length != 1) return;
    const container = containers.value[0];
    for (const entry of entries) {
      if (entry.isIntersecting) {
        const time = entry.target.getAttribute("data-time");
        if (time) {
          const date = new Date(parseInt(time));
          if (+date === +previousDate) break;
          previousDate = date;
          const diff = new Date().getTime() - container.created.getTime();
          progress.value = (date.getTime() - container.created.getTime()) / diff;
          currentDate.value = date;
          break;
        }
      }
    }
  },
  {
    rootMargin: "-10% 0px -10% 0px",
    threshold: 1,
  },
);
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
       below is the whole point: hover beats the zebra, and a level tint beats
       both, with its own (stronger) hover on top. */
    &:nth-child(odd) {
      background-color: color-mix(in oklab, var(--color-base-content) 4%, transparent);
    }

    &:hover {
      background-color: color-mix(in oklab, var(--color-base-content) 8%, transparent);
    }

    &[data-log-level="error"],
    &[data-log-level="fatal"] {
      background-color: color-mix(in oklab, var(--color-red) 9%, transparent);
    }

    &[data-log-level="error"]:hover,
    &[data-log-level="fatal"]:hover {
      background-color: color-mix(in oklab, var(--color-red) 15%, transparent);
    }

    &[data-log-level="warn"] {
      background-color: color-mix(in oklab, var(--color-orange) 8%, transparent);
    }

    &[data-log-level="warn"]:hover {
      background-color: color-mix(in oklab, var(--color-orange) 14%, transparent);
    }

    &.log-permalink-target {
      @apply bg-secondary/15;
      animation: log-permalink-pulse 1.4s ease-out;
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
