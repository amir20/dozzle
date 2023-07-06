import { type ComputedRef, type Ref } from "vue";
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
import { Container } from "@/models/Container";

function parseMessage(data: string): LogEntry<string | JSONObject> {
  const e = JSON.parse(data, (key, value) => {
    if (typeof value === "string") {
      return encodeXML(value);
    }
    return value;
  }) as LogEvent;
  return asLogEntry(e);
}

type LogStreamConfig = {
  stdout: boolean;
  stderr: boolean;
};

export function useLogStream(container: ComputedRef<Container>, streamConfig: LogStreamConfig) {
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

    const params = {
      lastEventId,
    } as { lastEventId: string; stdout?: string; stderr?: string };

    if (streamConfig.stdout) {
      params.stdout = "1";
    }
    if (streamConfig.stderr) {
      params.stderr = "1";
    }

    es = new EventSource(
      `${config.base}/api/logs/stream/${container.value.host}/${container.value.id}?${new URLSearchParams(
        params,
      ).toString()}`,
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

    const params = {
      from: from.toISOString(),
      to: to.toISOString(),
    } as { from: string; to: string; stdout?: string; stderr?: string };

    if (streamConfig.stdout) {
      params.stdout = "1";
    }
    if (streamConfig.stderr) {
      params.stderr = "1";
    }

    const logs = await (
      await fetch(
        `${config.base}/api/logs/${container.value.host}/${container.value.id}?${new URLSearchParams(
          params,
        ).toString()}`,
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

  onUnmounted(() => {
    if (es) {
      es.close();
    }
  });

  watch(
    () => container.value.id,
    () => connect(),
    { immediate: true },
  );

  watch(streamConfig, () => connect());

  return { ...$$({ messages }), loadOlderLogs };
}
