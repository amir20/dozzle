<template>
  <ul
    ref="list"
    class="events group py-4"
    :class="{ 'disable-wrap': !softWrap, [size]: true, compact }"
    v-if="messages.length > 0"
  >
    <li
      v-for="item in messages"
      :key="item.id"
      :data-key="item.id"
      :class="{ 'border border-secondary': toRaw(item) === toRaw(lastSelectedItem) }"
      class="group/entry"
    >
      <component
        :is="item.getComponent()"
        :log-entry="item"
        :visible-keys="visibleKeys"
        :show-container-name="showContainerName"
      />
    </li>
  </ul>
</template>

<script lang="ts" setup>
import { type JSONObject, LogEntry } from "@/models/LogEntry";

const { loading } = useScrollContext();

const { messages } = defineProps<{
  messages: LogEntry<string | JSONObject>[];
  visibleKeys: string[][];
  lastSelectedItem: LogEntry<string | JSONObject> | undefined;
  showContainerName: boolean;
}>();

watchEffect(() => {
  loading.value = messages.length === 0;
});
const list = ref<HTMLElement>();

useMutationObserver(
  list,
  (mutations) => {
    for (const mutation of mutations) {
      if (mutation.type === "childList") {
        const addedNodes = Array.from(mutation.addedNodes);
        for (const node of addedNodes) {
          if (node instanceof HTMLElement) {
            observer.observe(node);
          }
        }

        const removedNodes = Array.from(mutation.removedNodes);
        for (const node of removedNodes) {
          if (node instanceof HTMLElement) {
            observer.unobserve(node);
          }
        }
      }
    }
  },
  {
    childList: true,
  },
);

watchOnce(
  () => messages.length,
  () =>
    nextTick(() => {
      if (list.value) {
        Array.from(list.value.children).forEach((child) => observer.observe(child));
      }
    }),
);
const observer = new IntersectionObserver(
  (entries) => {
    for (const entry of entries) {
      if (entry.isIntersecting) {
        console.log("intersecting", entry);
      }
    }
  },
  {
    rootMargin: "0px 0px -90% 0px",
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
