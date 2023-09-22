<template>
  <ul class="events group py-4" :class="{ 'disable-wrap': !softWrap, [size]: true }">
    <li
      v-for="(item, index) in filtered"
      :key="item.id"
      :data-key="item.id"
      :class="{ 'border border-secondary': toRaw(item) === toRaw(lastSelectedItem) }"
      class="flex break-words px-4 py-1 last:snap-end odd:bg-base-lighter/30"
    >
      <a
        class="btn btn-ghost tooltip-primary tooltip btn-sm tooltip-right mr-4 flex self-start font-sans font-normal normal-case text-secondary hover:text-secondary-focus"
        v-show="isSearching()"
        data-tip="Jump to Context"
        @click="handleJumpLineSelected($event, item)"
        :href="`#${item.id}`"
      >
        <ic:sharp-find-in-page />
      </a>
      <component :is="item.getComponent()" :log-entry="item" :visible-keys="visibleKeys.value"></component>
    </li>
  </ul>
</template>

<script lang="ts" setup>
import { type ComputedRef, toRaw } from "vue";
import { useRouteHash } from "@vueuse/router";
import { Container } from "@/models/Container";
import { type JSONObject, LogEntry } from "@/models/LogEntry";

const props = defineProps<{
  messages: LogEntry<string | JSONObject>[];
}>();

let visibleKeys = persistentVisibleKeys(inject("container") as ComputedRef<Container>);

const { filteredPayload } = useVisibleFilter(visibleKeys);
const { filteredMessages, resetSearch, isSearching } = useSearchFilter();

const { messages } = toRefs(props);
const visible = filteredPayload(messages);
const filtered = filteredMessages(visible);

let lastSelectedItem: LogEntry<string | JSONObject> | undefined = $ref(undefined);

function handleJumpLineSelected(e: Event, item: LogEntry<string | JSONObject>) {
  lastSelectedItem = item;
  resetSearch();
}

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
}
</style>
