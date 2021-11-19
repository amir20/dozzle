<template>
  <ul class="events" :class="size">
    <li v-for="item in filtered" :key="item.key" :data-event="item.event">
      <span class="date" v-if="showTimestamp"> <relative-time :date="item.date"></relative-time></span>
      <span class="text" v-html="colorize(item.message)"></span>
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
const filtered = useSearchFilter().filteredMessages(messages);
</script>
<style scoped lang="scss">
.events {
  padding: 1em;
  font-family: SFMono-Regular, Consolas, Liberation Mono, monaco, Menlo, monospace;

  & > li {
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
