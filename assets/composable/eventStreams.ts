import { type Ref } from "vue";
import { encodeXML } from "entities";
import debounce from "lodash.debounce";
import {
  type LogEvent,
  type JSONObject,
  LogEntry,
  asLogEntry,
  ContainerEventLogEntry,
  SkippedLogsEntry,
} from "@/models/LogEntry";
import { Service, Stack } from "@/models/Stack";
import { Container, GroupedContainers } from "@/models/Container";

function parseMessage(data: string): LogEntry<string | JSONObject> {
  const e = JSON.parse(data, (key, value) => {
    if (typeof value === "string") {
      return encodeXML(value);
    }
    return value;
  }) as LogEvent;
  return asLogEntry(e);
}

export function useContainerStream(container: Ref<Container>): LogStreamSource {
  const { streamConfig } = useLoggingContext();

  const url = computed(() => {
    const params = Object.entries(toValue(streamConfig))
      .filter(([, value]) => value)
      .reduce((acc, [key]) => ({ ...acc, [key]: "1" }), {});
    return withBase(
      `/api/hosts/${container.value.host}/containers/${container.value.id}/logs/stream?${new URLSearchParams(params).toString()}`,
    );
  });

  const loadMoreUrl = computed(() => {
    const params = Object.entries(toValue(streamConfig))
      .filter(([, value]) => value)
      .reduce((acc, [key]) => ({ ...acc, [key]: "1" }), {});
    return withBase(
      `/api/hosts/${container.value.host}/containers/${container.value.id}/logs?${new URLSearchParams(params).toString()}`,
    );
  });

  return useLogStream(url, loadMoreUrl);
}

export function useStackStream(stack: Ref<Stack>): LogStreamSource {
  const { streamConfig } = useLoggingContext();

  const url = computed(() => {
    const params = Object.entries(toValue(streamConfig))
      .filter(([, value]) => value)
      .reduce((acc, [key]) => ({ ...acc, [key]: "1" }), {});
    return withBase(`/api/stacks/${stack.value.name}/logs/stream?${new URLSearchParams(params).toString()}`);
  });

  return useLogStream(url);
}

export function useGroupedStream(group: Ref<GroupedContainers>): LogStreamSource {
  const { streamConfig } = useLoggingContext();

  const url = computed(() => {
    const params = Object.entries(toValue(streamConfig))
      .filter(([, value]) => value)
      .reduce((acc, [key]) => ({ ...acc, [key]: "1" }), {});
    return withBase(`/api/groups/${group.value.name}/logs/stream?${new URLSearchParams(params).toString()}`);
  });

  return useLogStream(url);
}

export function useMergedStream(containers: Ref<Container[]>): LogStreamSource {
  const { streamConfig } = useLoggingContext();

  const url = computed(() => {
    const params = [
      ...Object.entries(toValue(streamConfig)).map(([key, value]) => [key, value ? "1" : "0"]),
      ...containers.value.map((c) => ["id", c.id]),
    ];

    return withBase(
      `/api/hosts/${containers.value[0].host}/logs/mergedStream?${new URLSearchParams(params).toString()}`,
    );
  });

  return useLogStream(url);
}

export function useServiceStream(service: Ref<Service>): LogStreamSource {
  const { streamConfig } = useLoggingContext();

  const url = computed(() => {
    const params = Object.entries(toValue(streamConfig))
      .filter(([, value]) => value)
      .reduce((acc, [key]) => ({ ...acc, [key]: "1" }), {});
    return withBase(`/api/services/${service.value.name}/logs/stream?${new URLSearchParams(params).toString()}`);
  });

  return useLogStream(url);
}

export type LogStreamSource = ReturnType<typeof useLogStream>;

function useLogStream(url: Ref<string>, loadMoreUrl?: Ref<string>) {
  const messages: Ref<LogEntry<string | JSONObject>[]> = shallowRef([]);
  let buffer: LogEntry<string | JSONObject>[] = $ref([]);
  const scrollingPaused = $ref(inject("scrollingPaused") as Ref<boolean>);

  function flushNow() {
    if (messages.value.length + buffer.length > config.maxLogs) {
      if (scrollingPaused) {
        console.log("Skipping ", buffer.length, " log items");
        if (messages.value.at(-1) instanceof SkippedLogsEntry) {
          const lastEvent = messages.value.at(-1) as SkippedLogsEntry;
          const lastItem = buffer.at(-1) as LogEntry<string | JSONObject>;
          lastEvent.addSkippedEntries(buffer.length, lastItem);
        } else {
          const firstItem = buffer.at(0) as LogEntry<string | JSONObject>;
          const lastItem = buffer.at(-1) as LogEntry<string | JSONObject>;

          messages.value = [...messages.value, new SkippedLogsEntry(new Date(), buffer.length, firstItem, lastItem)];
        }
        buffer = [];
      } else {
        messages.value = [...messages.value, ...buffer].slice(-config.maxLogs);
        buffer = [];
        console.log("Trimming logs to ", messages.value.length, " items");
      }
    } else {
      if (messages.value.length == 0) {
        console.log("Initial sort of logs");
        // sort the buffer the very first time because of multiple logs in parallel
        buffer.sort((a, b) => a.date.getTime() - b.date.getTime());
      }
      messages.value = [...messages.value, ...buffer];
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
    messages.value = [];
    buffer = [];
  }

  function connect({ clear } = { clear: true }) {
    close();

    if (clear) {
      clearMessages();
    }

    es = new EventSource(url.value);

    es.addEventListener("container-event", (e) => {
      const event = JSON.parse((e as MessageEvent).data) as { actorId: string; name: string };
      const containerEvent = new ContainerEventLogEntry(
        event.name == "container-started" ? "Container started" : "Container stopped",
        event.actorId,
        new Date(),
        event.name as "container-stopped" | "container-started",
      );
      buffer.push(containerEvent);

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

  let fetchingInProgress = false;

  async function loadOlderLogs() {
    if (!loadMoreUrl) return;
    if (fetchingInProgress) return;

    const to = messages.value[0].date;
    const last = messages.value[Math.min(messages.value.length - 1, 300)].date;
    const delta = to.getTime() - last.getTime();
    const from = new Date(to.getTime() + delta);

    const abortController = new AbortController();
    const signal = abortController.signal;
    fetchingInProgress = true;
    try {
      const stopWatcher = watchOnce(url, () => abortController.abort("stream changed"));
      const logs = await (
        await fetch(
          `${loadMoreUrl.value}&${new URLSearchParams({ from: from.toISOString(), to: to.toISOString() }).toString()}`,
          { signal },
        )
      ).text();
      stopWatcher();

      if (logs && signal.aborted === false) {
        const newMessages = logs
          .trim()
          .split("\n")
          .map((line) => parseMessage(line));
        messages.value = [...newMessages, ...messages.value];
      }
    } catch (e) {
      console.error("Error loading older logs", e);
    } finally {
      fetchingInProgress = false;
    }
  }

  onScopeDispose(() => close());

  const isLoadingMore = () => fetchingInProgress;

  return { ...$$({ messages }), loadOlderLogs, isLoadingMore };
}
