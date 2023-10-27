import { type Ref } from "vue";
import { encodeXML } from "entities";
import debounce from "lodash.debounce";
import {
  type LogEvent,
  type JSONObject,
  LogEntry,
  asLogEntry,
  DockerEventLogEntry,
  SkippedLogsEntry,
} from "@/models/LogEntry";

function parseMessage(data: string): LogEntry<string | JSONObject> {
  const e = JSON.parse(data, (key, value) => {
    if (typeof value === "string") {
      return encodeXML(value);
    }
    return value;
  }) as LogEvent;
  return asLogEntry(e);
}

export function useLogStream() {
  const { container, streamConfig } = useContainerContext();
  let messages: LogEntry<string | JSONObject>[] = $ref([]);
  let buffer: LogEntry<string | JSONObject>[] = $ref([]);
  const scrollingPaused = $ref(inject("scrollingPaused") as Ref<boolean>);
  let containerId = container.value.id;

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

  function close() {
    if (es) {
      es.close();
      console.debug(`EventSource closed for ${containerId}`);
      es = null;
    }
  }

  function clearMessages() {
    flushBuffer.cancel();
    messages = [];
    buffer = [];
    console.debug(`Clearing messages for ${containerId}`);
  }

  function connect({ clear } = { clear: true }) {
    close();

    if (clear) {
      clearMessages();
    }

    const params = Object.entries(streamConfig)
      .filter(([, value]) => value)
      .reduce((acc, [key]) => ({ ...acc, [key]: "1" }), {});

    containerId = container.value.id;

    console.debug(`Connecting to ${containerId} with params`, params);

    es = new EventSource(
      withBase(`/api/logs/stream/${container.value.host}/${containerId}?${new URLSearchParams(params).toString()}`),
    );
    es.addEventListener("container-stopped", () => {
      close();
      buffer.push(new DockerEventLogEntry("Container stopped", new Date(), "container-stopped"));

      flushBuffer();
      flushBuffer.flush();
    });
    es.onmessage = (e) => {
      if (e.data) {
        buffer.push(parseMessage(e.data));
        flushBuffer();
      }
    };
    es.onerror = () => clearMessages();
  }

  async function loadOlderLogs({ beforeLoading, afterLoading } = { beforeLoading: () => {}, afterLoading: () => {} }) {
    if (messages.length < 300) return;

    beforeLoading();
    const to = messages[0].date;
    const last = messages[299].date;
    const delta = to.getTime() - last.getTime();
    const from = new Date(to.getTime() + delta);

    const params = Object.entries(streamConfig)
      .filter(([, value]) => value)
      .reduce((acc, [key]) => ({ ...acc, [key]: "1" }), { from: from.toISOString(), to: to.toISOString() });

    const logs = await (
      await fetch(
        withBase(`/api/logs/${container.value.host}/${containerId}?${new URLSearchParams(params).toString()}`),
      )
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
    },
  );

  onUnmounted(() => close());

  watch(
    () => container.value.id,
    () => connect(),
    { immediate: true },
  );

  watch(streamConfig, () => connect());

  return { ...$$({ messages }), loadOlderLogs };
}
