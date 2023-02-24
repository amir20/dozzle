import { type ComputedRef, type Ref } from "vue";
import debounce from "lodash.debounce";
import {
  type LogEvent,
  type JSONObject,
  LogEntry,
  asLogEntry,
  DockerEventLogEntry,
  SkippedLogsEntry,
} from "@/models/LogEntry";
import { Container } from "@/models/Container";

function parseMessage(data: string): LogEntry<string | JSONObject> {
  const e = JSON.parse(data) as LogEvent;
  return asLogEntry(e);
}

export function useLogStream(container: ComputedRef<Container>) {
  let messages: LogEntry<string | JSONObject>[] = $ref([]);
  let buffer: LogEntry<string | JSONObject>[] = $ref([]);
  const scrollingPaused = $ref(inject("scrollingPaused") as Ref<boolean>);

  function flushNow() {
    if (messages.length > config.maxLogs) {
      if (scrollingPaused) {
        console.log("Skipping ", buffer.length, " log items");
        if (messages.at(-1) instanceof SkippedLogsEntry) {
          const lastEvent = messages.at(-1) as SkippedLogsEntry;
          const lastItem = buffer.at(-1) as LogEntry<string | JSONObject>;
          lastEvent.addSkippedEntries(buffer.length, lastItem);
        } else {
          const firstItem = buffer.at(0) as LogEntry<string | JSONObject>;
          const lastItem = buffer.at(-1) as LogEntry<string | JSONObject>;
          messages.push(new SkippedLogsEntry(new Date(), buffer.length, firstItem, lastItem));
        }
        buffer = [];
      } else {
        messages.push(...buffer);
        buffer = [];
        messages = messages.slice(-config.maxLogs);
      }
    } else {
      messages.push(...buffer);
      buffer = [];
    }
  }
  const flushBuffer = debounce(flushNow, 250, { maxWait: 1000 });
  let es: EventSource | null = null;
  let lastEventId = "";

  function connect({ clear } = { clear: true }) {
    es?.close();

    if (clear) {
      flushBuffer.cancel();
      messages = [];
      buffer = [];
      lastEventId = "";
    }

    es = new EventSource(
      `${config.base}/api/logs/stream?id=${container.value.id}&lastEventId=${lastEventId}&host=${sessionHost.value}`
    );
    es.addEventListener("container-stopped", () => {
      es?.close();
      es = null;
      buffer.push(new DockerEventLogEntry("Container stopped", new Date(), "container-stopped"));

      flushBuffer();
      flushBuffer.flush();
    });
    es.addEventListener("error", (e) => console.error("EventSource failed: " + JSON.stringify(e)));
    es.onmessage = (e) => {
      lastEventId = e.lastEventId;
      if (e.data) {
        buffer.push(parseMessage(e.data));
        flushBuffer();
      }
    };
  }

  async function loadOlderLogs({ beforeLoading, afterLoading } = { beforeLoading: () => {}, afterLoading: () => {} }) {
    if (messages.length < 300) return;

    beforeLoading();
    const to = messages[0].date;
    const last = messages[299].date;
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
      messages.unshift(...newMessages);
    }
    afterLoading();
  }

  watch(
    () => container.value.state,
    (newValue, oldValue) => {
      console.log("LogEventSource: container changed", newValue, oldValue);
      if (newValue == "running" && newValue != oldValue) {
        buffer.push(new DockerEventLogEntry("Container started", new Date(), "container-started"));
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
    () => connect(),
    { immediate: true }
  );

  return { ...$$({ messages }), loadOlderLogs };
}
