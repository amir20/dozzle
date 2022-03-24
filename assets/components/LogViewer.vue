<template>
  <ul class="events" :class="size" ref="events">
    <li
      v-for="(item, index) in filtered"
      :key="item.key"
      :data-event="item.event"
      :data-key="item.key"
      :class="item.selected ? 'selected' : ''"
    >
      <div class="line-options" v-if="isSearching()">
        <dropdown-menu :class="{ 'is-last': index === filtered.length - 1 }" class="is-top minimal">
          <a class="dropdown-item" @click="jumpToLine">
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
        <span class="text" v-html="colorize(item.message)"></span>
      </div>
    </li>
  </ul>
</template>

<script lang="ts" setup>
import { PropType, ref, toRefs } from "vue";

import { size, showTimestamp } from "@/composables/settings";
import RelativeTime from "./RelativeTime.vue";
import AnsiConvertor from "ansi-to-html";
import { LogEntry } from "@/types/LogEntry";
import { useSearchFilter } from "@/composables/search";
import { useContainerStore } from "@/stores/container";
import { storeToRefs } from "pinia";

const props = defineProps({
  messages: {
    type: Array as PropType<LogEntry[]>,
    required: true,
  },
});

const ansiConvertor = new AnsiConvertor({ escapeXML: true });
const colorize = (value: string) =>
  ansiConvertor.toHtml(value).replace("&lt;mark&gt;", "<mark>").replace("&lt;/mark&gt;", "</mark>");
const { messages } = toRefs(props);
const { filteredMessages, searchFilter, resetSearch, isSearching } = useSearchFilter();
const store = useContainerStore();
const { activeContainers } = storeToRefs(store);
const filtered = filteredMessages(messages);
const events = ref(null);
const jumpToLine = async (e) => {
  const line = e.target.closest("li");
  if (line.tagName !== "LI") {
    return;
  }
  resetSearch();
  for (const item of messages.value) {
    item.selected = false;
    if (item.key === line.dataset.key) {
      item.selected = true;
    }
  }
  // when in split pane mode - scroll the pane element, when not in split pane mode - scroll the window element
  const elemToScroll = activeContainers.value.length > 0 ? events.value.closest("main") : window;
  // TODO have pane scroll to the line without timing hacks, this will require telling the scrollable view when we're jumping to context, so it can stop scrolling to bottom temporarily
  await new Promise((resolve) => setTimeout(resolve, 10));
  elemToScroll.scrollTo(0, 0);
  await new Promise((resolve) => setTimeout(resolve, 1000));
  elemToScroll.scrollTo(0, line.offsetTop - 200);
};
</script>
<style scoped lang="scss">
.events {
  padding: 1em;
  font-family: SFMono-Regular, Consolas, Liberation Mono, monaco, Menlo, monospace;
  overflow: hidden;

  & > li {
    display: flex;
    word-wrap: break-word;
    line-height: 130%;
    &:last-child {
      scroll-snap-align: end;
      scroll-margin-block-end: 5rem;
    }
    &[data-event="container-stopped"] {
      color: #f14668;
    }
    &[data-event="container-started"] {
      color: hsl(141, 53%, 53%);
    }
    &.selected {
      background-color: var(--menu-item-active-background-color);
      color: black;
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
