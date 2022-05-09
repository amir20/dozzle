<template>
  <infinite-loader :onLoadMore="loadOlderLogs" :enabled="messages.length > 100"></infinite-loader>
  <slot :messages="messages"></slot>
</template>

<script lang="ts" setup>
import { toRefs, ref, watch, onUnmounted } from "vue";
import debounce from "lodash.debounce";

import { LogEntry, LogEvent } from "@/types/LogEntry";
import InfiniteLoader from "./InfiniteLoader.vue";
import config from "@/stores/config";
import { useContainerStore } from "@/stores/container";

const props = defineProps({
  id: {
    type: String,
    required: true,
  },
});

const { id } = toRefs(props);
const emit = defineEmits(["loading-more"]);
const store = useContainerStore();
const container = store.currentContainer(id);

const messages = ref<LogEntry[]>([]);
const buffer = ref<LogEntry[]>([]);

function flushNow() {
  messages.value.push(...buffer.value);
  buffer.value = [];
}

const flushBuffer = debounce(flushNow, 250, { maxWait: 1000 });

let es: EventSource | null = null;
let lastEventId = "";

function connect({ clear } = { clear: true }) {
  es?.close();

  if (clear) {
    flushBuffer.cancel();
    messages.value = [];
    buffer.value = [];
    lastEventId = "";
  }

  es = new EventSource(`${config.base}/api/logs/stream?id=${props.id}&lastEventId=${lastEventId}`);
  es.addEventListener("container-stopped", () => {
    es?.close();
    es = null;
    buffer.value.push({
      event: "container-stopped",
      message: "Container stopped",
      date: new Date(),
      key: new Date().toString(),
    });
    flushBuffer();
    flushBuffer.flush();
  });
  es.addEventListener("error", (e) => console.error("EventSource failed: " + JSON.stringify(e)));
  es.onmessage = (e) => {
    lastEventId = e.lastEventId;
    if (e.data) {
      buffer.value.push(parseMessage(e.data));
      flushBuffer();
    }
  };
}

async function loadOlderLogs() {
  if (messages.value.length < 300) return;

  emit("loading-more", true);
  const to = messages.value[0].date;
  const last = messages.value[299].date;
  const delta = to.getTime() - last.getTime();
  const from = new Date(to.getTime() + delta);
  const logs = await (
    await fetch(`${config.base}/api/logs?id=${props.id}&from=${from.toISOString()}&to=${to.toISOString()}`)
  ).text();
  if (logs) {
    const newMessages = logs
      .trim()
      .split("\n")
      .map((line) => parseMessage(line));
    messages.value.unshift(...newMessages);
  }
  emit("loading-more", false);
}

function parseMessage(data: string): LogEntry {
  const e = JSON.parse(data) as LogEvent;

  const key = e.ts.toString();
  const date = new Date(e.ts);
  const message = e.m;
  return { key, date, message };
}

watch(
  () => container.value.state,
  (newValue, oldValue) => {
    console.log("LogEventSource: container changed", newValue, oldValue);
    if (newValue == "running" && newValue != oldValue) {
      buffer.value.push({
        event: "container-started",
        message: "Container started",
        date: new Date(),
        key: new Date().toString(),
      });
      connect({ clear: false });
    }
  }
);

onUnmounted(() => {
  if (es) {
    es.close();
  }
});

connect();
watch(id, () => connect());

defineExpose({
  clear: () => (messages.value = []),
});
</script>
