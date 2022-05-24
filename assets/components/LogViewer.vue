<template>
  <ul class="events" ref="events" :class="{ 'disable-wrap': !softWrap, [size]: true }">
    <li
      v-for="(item, index) in filtered"
      :key="item.key"
      :data-key="item.key"
      :data-event="item.event"
      :class="{ selected: item.selected }"
    >
      <div class="line-options" v-show="isSearching()">
        <dropdown-menu :class="{ 'is-last': index === filtered.length - 1 }" class="is-top minimal">
          <a class="dropdown-item" @click="handleJumpLineSelected($event, item)" :href="`#${item.key}`">
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
        <span class="text" v-html="colorize(item.message)" v-if="item.message"></span>
        <JSONPayload :payload="item.payload" v-else></JSONPayload>
      </div>
    </li>
  </ul>
</template>

<script lang="ts" setup>
import { PropType, ref, toRefs, watch } from "vue";
import { useRouteHash } from "@vueuse/router";
import { size, showTimestamp, softWrap } from "@/composables/settings";
import RelativeTime from "./RelativeTime.vue";
import AnsiConvertor from "ansi-to-html";
import { LogEntry } from "@/types/LogEntry";
import { useSearchFilter } from "@/composables/search";
import JSONPayload from "./JSONPayload.vue";

const props = defineProps({
  messages: {
    type: Array as PropType<LogEntry[]>,
    required: true,
  },
});

const ansiConvertor = new AnsiConvertor({ escapeXML: true });
const { filteredMessages, resetSearch, markSearch, isSearching } = useSearchFilter();
const colorize = (value: string) => markSearch(ansiConvertor.toHtml(value));
const { messages } = toRefs(props);
const filtered = filteredMessages(messages);
const events = ref<HTMLElement>();
let lastSelectedItem: LogEntry | undefined = undefined;
function handleJumpLineSelected(e: Event, item: LogEntry) {
  if (lastSelectedItem) {
    lastSelectedItem.selected = false;
  }
  lastSelectedItem = item;
  item.selected = true;
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
    .line,
    .text {
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
    &[data-event="container-stopped"] {
      color: #f14668;
    }
    &[data-event="container-started"] {
      color: hsl(141, 53%, 53%);
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

.text {
  white-space: pre-wrap;
  &::before {
    content: " ";
  }
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
