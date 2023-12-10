<template>
  <ul class="events group py-4" :class="{ 'disable-wrap': !softWrap, [size]: true }">
    <li
      v-for="item in filtered"
      :key="item.id"
      :data-key="item.id"
      :class="{ 'border border-secondary': toRaw(item) === toRaw(lastSelectedItem) }"
      @jump-context="console.log('jump context event triggered')"
      class="group/entry"
    >
      <component :is="item.getComponent()" :log-entry="item" :visible-keys="visibleKeys" />
    </li>
  </ul>
</template>

<script lang="ts" setup>
import { toRaw } from "vue";
import { useRouteHash } from "@vueuse/router";

import { type JSONObject, LogEntry } from "@/models/LogEntry";

const props = defineProps<{
  messages: LogEntry<string | JSONObject>[];
}>();

const { container } = useContainerContext();

const visibleKeys = persistentVisibleKeys(container);

const { filteredPayload } = useVisibleFilter(visibleKeys);
const { filteredMessages } = useSearchFilter();

const { messages } = toRefs(props);
const visible = filteredPayload(messages);
const filtered = filteredMessages(visible);

const { lastSelectedItem } = useLogSearchContext();
const routeHash = useRouteHash();
watch(
  routeHash,
  (hash) => {
    if (hash) {
      document.querySelector(`[data-key="${hash.substring(1)}"]`)?.scrollIntoView({ block: "center" });
    }
  },
  { immediate: true, flush: "post" },
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
    @apply flex break-words px-4 py-1 last:snap-end odd:bg-gray-400/[0.07];
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
}
</style>
