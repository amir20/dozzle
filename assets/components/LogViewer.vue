<template>
  <ul class="events" :class="size">
    <li
      v-for="item in filtered"
      :key="item.key"
      :data-event="item.event"
      :data-key="item.key"
      :class="item.selected ? 'selected' : ''"
    >
      <div>
        <span class="date" v-if="showTimestamp"> <relative-time :date="item.date"></relative-time></span>
        <span class="text" v-html="colorize(item.message)"></span>
      </div>
      <div class="line-options" v-if="isSearching()">
        <o-button variant="primary" inverted size="small" type="button" class="jump" @click="jumpToLine">
          <span class="icon">
            <cil-find-in-page />
          </span>
        </o-button>
      </div>
    </li>
  </ul>
</template>

<script lang="ts" setup>
import { PropType, toRefs } from "vue";

import { size, showTimestamp } from "@/composables/settings";
import RelativeTime from "./RelativeTime.vue";
import AnsiConvertor from "ansi-to-html";
import { LogEntry } from "@/types/LogEntry";
import { useSearchFilter } from "@/composables/search";

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
const filtered = filteredMessages(messages);
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
  // TODO prevent ScrollableView from automatically sticking to the bottom when closing search and jumping to context (ScrollableView.vue:47)
  // if that could be achieved, then these timing hacks can be removed
  await new Promise((resolve) => setTimeout(resolve, 10));
  window.scrollTo(0, 0);
  await new Promise((resolve) => setTimeout(resolve, 1000));
  window.scrollTo(0, line.offsetTop - 200);
};
</script>
<style scoped lang="scss">
.events {
  padding: 1em;
  font-family: SFMono-Regular, Consolas, Liberation Mono, monaco, Menlo, monospace;

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
      background-color: white;
      color: black;
    }
    &.selected > .date {
      background-color: white;
    }
    & > .line-options {
      flex: 1;
      display: flex;
      flex-direction: row-reverse;
      & > .jump {
        cursor: pointer;
      }
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

.date {
  background-color: #262626;
  color: #258ccd;

  [data-theme="light"] & {
    background-color: #f0f0f0;
    color: #009900;
    padding-left: 5px;
    padding-right: 5px;
  }
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
