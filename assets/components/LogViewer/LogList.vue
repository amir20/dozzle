<template>
  <ul class="group pt-4" :class="{ 'disable-wrap': !softWrap, [size]: true, compact }" data-logs>
    <li
      v-for="item in rows"
      ref="list"
      :key="item.id"
      :id="item.id.toString()"
      :data-time="item.date.getTime()"
      class="group/entry"
      :class="{ 'log-permalink-target': permalinkLogId === item.id.toString() }"
    >
      <component :is="item.getComponent()" :log-entry="item" />
    </li>
  </ul>
</template>

<script lang="ts" setup>
import type { LogEntry, LogMessage } from "@/models/LogEntry";
// TEMPORARY dev harness — remove before merge.
import { AlertLogEntry } from "@/models/LogEntry";

const { progress, currentDate } = useScrollContext();

const { messages } = defineProps<{
  messages: LogEntry<LogMessage>[];
}>();

const { containers } = useLoggingContext();

// TEMPORARY dev harness — remove before merge.
// Appends a synthetic alert as the LAST entry when ?alertPreview=1 is set, so
// the row can be inspected in its real LogItem/LogList context without
// scrolling back hours to find a real one.
const previewAlert = computed(() => {
  if (typeof location === "undefined" || !new URLSearchParams(location.search).has("alertPreview")) return null;
  const now = Date.now();
  return new AlertLogEntry(
    {
      alertId: 999999,
      containerId: containers.value[0]?.id ?? "preview",
      hostId: "preview",
      ts: now * 1_000_000,
      headline: "Container restarting on a failed health check",
      level: "error",
      eventCount: 35,
      suppressedCount: 34,
      containerCount: 3,
      summary: "Container `worker` running on `orbstack` exited 137 (OOM) 35 times in 14 minutes.",
      investigation:
        "Health check has failed 35 times in 14 minutes. The container exits 137 each time, which is an OOM kill — the memory limit is 256Mi and RSS peaks at 251Mi just before each exit.",
      createdAt: now * 1_000_000,
      lastActivityAt: (now + 828_000) * 1_000_000,
      isOrigin: true,
      url: "https://cloud.dozzle.dev/alerts/999999",
    },
    new Date(now),
  );
});

const rows = computed(() => (previewAlert.value ? [...messages, previewAlert.value] : messages));

const route = useRoute();
const permalinkLogId = computed(() => (typeof route.query.logId === "string" ? route.query.logId : ""));

const list = ref<HTMLElement[]>([]);

let previousDate = new Date();
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
  font-family:
    ui-monospace,
    SFMono-Regular,
    SF Mono,
    Consolas,
    Liberation Mono,
    monaco,
    Menlo,
    monospace;

  > li {
    @apply flex px-2 py-1 break-words last:snap-end odd:bg-gray-400/[0.07] md:px-4;
    &:last-child {
      scroll-margin-block-end: 5rem;
    }

    &.log-permalink-target {
      @apply bg-secondary/15 border-secondary -ml-1 border-l-4 pl-3;
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
