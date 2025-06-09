<template>
  <ul class="group pt-4" :class="{ 'disable-wrap': !softWrap, [size]: true, compact }" data-logs>
    <li
      v-for="item in messages"
      ref="list"
      :key="item.id"
      :id="item.id.toString()"
      :data-time="item.date.getTime()"
      class="group/entry"
    >
      <component :is="item.getComponent()" :log-entry="item" />
    </li>
  </ul>
</template>

<script lang="ts" setup>
import { type JSONObject, LogEntry } from "@/models/LogEntry";

const { progress, currentDate } = useScrollContext();

const { messages } = defineProps<{
  messages: LogEntry<string | JSONObject>[];
}>();

const { containers } = useLoggingContext();

const list = ref<HTMLElement[]>([]);

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
          const diff = new Date().getTime() - container.created.getTime();
          progress.value = (date.getTime() - container.created.getTime()) / diff;
          currentDate.value = date;
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
  }

  &.small {
    @apply text-[0.7em];
  }

  &.medium {
    @apply text-[0.8em];
  }

  &.large {
    @apply text-lg;
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
</style>
