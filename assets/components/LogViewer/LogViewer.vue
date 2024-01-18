<template>
  <ul class="events group py-4" :class="{ 'disable-wrap': !softWrap, [size]: true, compact }">
    <li
      v-for="item in messages"
      :key="item.id"
      :data-key="item.id"
      :class="{ 'border border-secondary': toRaw(item) === toRaw(lastSelectedItem) }"
      class="group/entry"
    >
      <component :is="item.getComponent()" :log-entry="item" :visible-keys="visibleKeys" />
    </li>
  </ul>
</template>

<script lang="ts" setup>
import { toRaw } from "vue";

import { type JSONObject, LogEntry } from "@/models/LogEntry";

defineProps<{
  messages: LogEntry<string | JSONObject>[];
  visibleKeys: string[][];
  lastSelectedItem: LogEntry<string | JSONObject> | undefined;
}>();
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
