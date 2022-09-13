<template>
  <ul class="events" ref="events" :class="{ 'disable-wrap': !softWrap, [size]: true }">
    <li
      v-for="(item, index) in filtered"
      :key="item.id"
      :data-key="item.id"
      :class="{ selected: toRaw(item) === toRaw(lastSelectedItem) }"
    >
      <div class="line-options" v-show="isSearching()">
        <dropdown-menu :class="{ 'is-last': index === filtered.length - 1 }" class="is-top minimal">
          <a class="dropdown-item" @click="handleJumpLineSelected($event, item)" :href="`#${item.id}`">
            <div class="level is-justify-content-start">
              <div class="level-left">
                <div class="level-item">
                  <cil-find-in-page class="mr-4" />
                </div>
              </div>
              <div class="level-right">
                <div class="level-item">Jump to Context</div>
              </div>
            </div>
          </a>
        </dropdown-menu>
      </div>
      <div class="line">
        <span class="date" v-if="showTimestamp"> <relative-time :date="item.date"></relative-time></span>
        <component :is="item.getComponent()" :log-entry="item" :visible-keys="visibleKeys.value"></component>
      </div>
    </li>
  </ul>
</template>

<script lang="ts" setup>
import { type ComputedRef, toRaw } from "vue";
import { useRouteHash } from "@vueuse/router";
import { type Container } from "@/types/Container";
import { type JSONObject, type LogEntry } from "@/models/LogEntry";

const props = defineProps<{
  messages: LogEntry<string | JSONObject>[];
}>();

const { messages } = toRefs(props);
let visibleKeys = persistentVisibleKeys(inject("container") as ComputedRef<Container>);

const { filteredPayload } = useVisibleFilter(visibleKeys);
const { filteredMessages, resetSearch, isSearching } = useSearchFilter();

const visible = filteredPayload(messages);
const filtered = filteredMessages(visible);

const events = ref<HTMLElement>();
let lastSelectedItem = ref<LogEntry<string | JSONObject>>();

function handleJumpLineSelected(e: Event, item: LogEntry<string | JSONObject>) {
  lastSelectedItem.value = item;
  resetSearch();
}

const routeHash = useRouteHash();
watch(
  routeHash,
  (hash) => {
    document.querySelector(`[data-key="${hash.substring(1)}"]`)?.scrollIntoView({ block: "center" });
  },
  { immediate: true, flush: "post" }
);
</script>
<style scoped lang="scss">
.events {
  padding: 1em 0;
  font-family: SFMono-Regular, Consolas, Liberation Mono, monaco, Menlo, monospace;

  &.disable-wrap {
    .line {
      white-space: nowrap;
    }
  }

  & > li {
    display: flex;
    word-wrap: break-word;
    padding: 0.2em 1em;
    &:last-child {
      scroll-snap-align: end;
      scroll-margin-block-end: 5rem;
    }
    &:nth-child(odd) {
      background-color: rgba(125, 125, 125, 0.08);
    }

    &.selected .date {
      background-color: var(--menu-item-active-background-color);

      color: var(--text-color);
    }
    &.selected > .date {
      background-color: white;
    }
    & > .line {
      margin: auto 0;
      width: 100%;
      display: flex;
    }
    & > .line-options {
      display: flex;
      flex-direction: row-reverse;
      margin-right: 1em;
    }
  }

  &.small {
    font-size: 60%;
  }

  &.medium {
    font-size: 80%;
  }

  &.large {
    font-size: 120%;
  }
}

@media (prefers-color-scheme: dark) {
  .date {
    background-color: #262626;
    color: #258ccd;
  }
}

[data-theme="dark"] {
  .date {
    background-color: #262626;
    color: #258ccd;
  }
}

@media (prefers-color-scheme: light) {
  .date {
    background-color: #f0f0f0;
    color: #009900;
  }
}

[data-theme="light"] {
  .date {
    background-color: #f0f0f0;
    color: #009900;
  }
}

.date {
  padding-left: 5px;
  padding-right: 5px;
  border-radius: 3px;
}

:deep(mark) {
  border-radius: 2px;
  background-color: var(--secondary-color);
  animation: pops 200ms ease-out;
  display: inline-block;
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
