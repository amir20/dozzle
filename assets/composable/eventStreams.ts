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

export function useContainerContextLogStream() {
  const { container, streamConfig } = useContainerContext();

  const url = computed(() => {
    const params = Object.entries(streamConfig)
      .filter(([, value]) => value)
      .reduce((acc, [key]) => ({ ...acc, [key]: "1" }), {});
    return withBase(
      `/api/hosts/${container.value.host}/containers/${container.value.id}/logs/stream?${new URLSearchParams(params).toString()}`,
    );
  });

  return useLogStream(url);
}

export function useStackContextLogStream() {
  const { stack, streamConfig } = useStackContext();

  const url = computed(() => {
    const params = Object.entries(streamConfig)
      .filter(([, value]) => value)
      .reduce((acc, [key]) => ({ ...acc, [key]: "1" }), {});
    return withBase(`/api/stacks/${stack.value.name}/logs/stream?${new URLSearchParams(params).toString()}`);
  });

  return useLogStream(url);
}

function useLogStream(url: Ref<string>) {
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

  function close() {
    if (es) {
      es.close();
      es = null;
    }
  }

  function clearMessages() {
    flushBuffer.cancel();
    messages = [];
    buffer = [];
    // console.debug(`Clearing messages for ${containerId}`);
  }

  function connect({ clear } = { clear: true }) {
    close();

    if (clear) {
      clearMessages();
    }

    es = new EventSource(url.value);

    es.addEventListener("container-stopped", () => {
      // TODO container id
      close();
      buffer.push(new DockerEventLogEntry("Container stopped", "123", new Date(), "container-stopped"));

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

  watch(url, () => connect(), { immediate: true });

  async function loadOlderLogs({ beforeLoading, afterLoading } = { beforeLoading: () => {}, afterLoading: () => {} }) {
    // if (messages.length < 300) return;

    // beforeLoading();
    // const to = messages[0].date;
    // const last = messages[299].date;
    // const delta = to.getTime() - last.getTime();
    // const from = new Date(to.getTime() + delta);

    // const params = Object.entries(streamConfig)
    //   .filter(([, value]) => value)
    //   .reduce((acc, [key]) => ({ ...acc, [key]: "1" }), { from: from.toISOString(), to: to.toISOString() });

    throw new Error("Not implemented");
    // const logs = await (
    //   await fetch(
    //     withBase(
    //       `/api/hosts/${container.value.host}/containers/${containerId}/logs?${new URLSearchParams(params).toString()}`,
    //     ),
    //   )
    // ).text();
    // if (logs) {
    //   const newMessages = logs
    //     .trim()
    //     .split("\n")
    //     .map((line) => parseMessage(line));
    //   messages.unshift(...newMessages);
    // }
    // afterLoading();
  }

  // watch(
  //   () => container.value.state,
  //   (newValue, oldValue) => {
  //     console.log("LogEventSource: container changed", newValue, oldValue);
  //     if (newValue == "running" && newValue != oldValue) {
  //       buffer.push(new DockerEventLogEntry("Container started", new Date(), "container-started"));
  //       connect({ clear: false });
  //     }
  //   },
  // );

  onScopeDispose(() => close());

  // watch(
  //   () => container.value.id,
  //   () => connect(),
  //   { immediate: true },
  // );

  // watch(streamConfig, () => connect());

  return { ...$$({ messages }), loadOlderLogs };
}
