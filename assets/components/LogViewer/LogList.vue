<template>
  <ul
    class="events group pt-4"
    :class="{ 'disable-wrap': !softWrap, [size]: true, compact }"
    v-if="messages.length > 0"
  >
    <li
      v-for="item in messages"
      ref="list"
      :key="item.id"
      :data-key="item.id"
      :data-time="item.date.getTime()"
      :class="{ 'border border-secondary': toRaw(item) === toRaw(lastSelectedItem) }"
      class="group/entry"
    >
      <component :is="item.getComponent()" :log-entry="item" :show-container-name="showContainerName" />
    </li>
  </ul>
</template>

<script lang="ts" setup>
import { type JSONObject, LogEntry } from "@/models/LogEntry";

const { loading, progress, currentDate } = useScrollContext();

const { messages } = defineProps<{
  messages: LogEntry<string | JSONObject>[];
  lastSelectedItem: LogEntry<string | JSONObject> | undefined;
  showContainerName: boolean;
}>();

watchEffect(() => {
  loading.value = messages.length === 0;
});

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
<style scoped lang="postcss">
.events {
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
    @apply flex break-words px-2 py-1 last:snap-end odd:bg-gray-400/[0.07] md:px-4;
    &:last-child {
      scroll-margin-block-end: 5rem;
    }

    .jump-context {
      @apply mr-2 flex items-center font-sans text-secondary;
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
  }
}
</style>
