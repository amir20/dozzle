<template>
  <ul class="group pt-4" :class="{ 'disable-wrap': !softWrap, [size]: true, compact }" data-logs>
    <li
      v-for="item in messages"
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

const { progress, currentDate } = useScrollContext();

const { messages } = defineProps<{
  messages: LogEntry<LogMessage>[];
}>();

const { containers } = useLoggingContext();

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
