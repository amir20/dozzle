import { ref, watch, onUnmounted, ComputedRef } from "vue";
import debounce from "lodash.debounce";

import { LogEntry, LogEvent } from "@/types/LogEntry";

import config from "@/stores/config";
import { Container } from "@/types/Container";

function parseMessage(data: string): LogEntry {
  const e = JSON.parse(data) as LogEvent;

  const key = e.ts.toString();
  const date = new Date(e.ts * 1000);
  return { key, date, message: e.m, payload: e.d };
}

export function useEventSource(container: ComputedRef<Container>) {
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

    es = new EventSource(`${config.base}/api/logs/stream?id=${container.value.id}&lastEventId=${lastEventId}`);
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

  async function loadOlderLogs({ beforeLoading, afterLoading } = { beforeLoading: () => {}, afterLoading: () => {} }) {
    if (messages.value.length < 300) return;

    beforeLoading();
    const to = messages.value[0].date;
    const last = messages.value[299].date;
    const delta = to.getTime() - last.getTime();
    const from = new Date(to.getTime() + delta);
    const logs = await (
      await fetch(`${config.base}/api/logs?id=${container.value.id}&from=${from.toISOString()}&to=${to.toISOString()}`)
    ).text();
    if (logs) {
      const newMessages = logs
        .trim()
        .split("\n")
        .map((line) => parseMessage(line));
      messages.value.unshift(...newMessages);
    }
    afterLoading();
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

  watch(
    () => container.value.id,
    () => connect()
  );

  return { connect, messages, loadOlderLogs };
}
