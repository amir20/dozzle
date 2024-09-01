import { ShallowRef, type Ref } from "vue";

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

const { isSearching, debouncedSearchFilter } = useSearchFilter();

function parseMessage(data: string): LogEntry<string | JSONObject> {
  const e = JSON.parse(data) as LogEvent;
  return asLogEntry(e);
}

export function useContainerStream(container: Ref<Container>): LogStreamSource {
  const url = computed(() =>
    withBase(`/api/hosts/${container.value.host}/containers/${container.value.id}/logs/stream`),
  );

  const loadMoreUrl = computed(() =>
    withBase(`/api/hosts/${container.value.host}/containers/${container.value.id}/logs`),
  );

  return useLogStream(url, loadMoreUrl);
}

export function useStackStream(stack: Ref<Stack>): LogStreamSource {
  const url = computed(() => withBase(`/api/stacks/${stack.value.name}/logs/stream`));
  return useLogStream(url);
}

export function useGroupedStream(group: Ref<GroupedContainers>): LogStreamSource {
  const url = computed(() => withBase(`/api/groups/${group.value.name}/logs/stream`));
  return useLogStream(url);
}

export function useMergedStream(containers: Ref<Container[]>): LogStreamSource {
  const url = computed(() => {
    const ids = containers.value.map((c) => ["id", c.id]).join(",");
    return withBase(`/api/hosts/${containers.value[0].host}/logs/mergedStream/${ids}`);
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
  const messages: ShallowRef<LogEntry<string | JSONObject>[]> = shallowRef([]);
  const buffer: ShallowRef<LogEntry<string | JSONObject>[]> = shallowRef([]);
  const { paused: scrollingPaused } = useScrollContext();

  function flushNow() {
    if (messages.value.length + buffer.value.length > config.maxLogs) {
      if (scrollingPaused.value === true) {
        if (messages.value.at(-1) instanceof SkippedLogsEntry) {
          const lastEvent = messages.value.at(-1) as SkippedLogsEntry;
          const lastItem = buffer.value.at(-1) as LogEntry<string | JSONObject>;
          lastEvent.addSkippedEntries(buffer.value.length, lastItem);
        } else {
          const firstItem = buffer.value.at(0) as LogEntry<string | JSONObject>;
          const lastItem = buffer.value.at(-1) as LogEntry<string | JSONObject>;

          messages.value = [
            ...messages.value,
            new SkippedLogsEntry(new Date(), buffer.value.length, firstItem, lastItem),
          ];
        }
        buffer.value = [];
      } else {
        if (buffer.value.length > config.maxLogs / 2) {
          messages.value = buffer.value.slice(-config.maxLogs / 2);
        } else {
          messages.value = [...messages.value, ...buffer.value].slice(-config.maxLogs);
        }
        buffer.value = [];
      }
    } else {
      if (messages.value.length == 0) {
        // sort the buffer the very first time because of multiple logs in parallel
        buffer.value.sort((a, b) => a.date.getTime() - b.date.getTime());
      }
      messages.value = [...messages.value, ...buffer.value];
      buffer.value = [];
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
    buffer.value = [];
  }

  const { streamConfig } = useLoggingContext();

  const params = computed(() => {
    const params = Object.entries(toValue(streamConfig))
      .filter(([, value]) => value)
      .reduce((acc, [key]) => ({ ...acc, [key]: "1" }), {} as Record<string, string>);

    if (isSearching.value) {
      params["filter"] = debouncedSearchFilter.value;
    }

    return params;
  });

  const urlWithParams = computed(() => withBase(`${url.value}?${new URLSearchParams(params.value).toString()}`));

  function connect({ clear } = { clear: true }) {
    close();
    if (clear) clearMessages();
    es = new EventSource(urlWithParams.value);
    es.addEventListener("container-event", (e) => {
      const event = JSON.parse((e as MessageEvent).data) as { actorId: string; name: string };
      const containerEvent = new ContainerEventLogEntry(
        event.name == "container-started" ? "Container started" : "Container stopped",
        event.actorId,
        new Date(),
        event.name as "container-stopped" | "container-started",
      );

      buffer.value = [...buffer.value, containerEvent];

      flushBuffer.flush();
    });
    es.onmessage = (e) => {
      if (e.data) {
        buffer.value = [...buffer.value, parseMessage(e.data)];
        flushBuffer();
      }
    };
    es.onerror = () => clearMessages();
  }

  watch(urlWithParams, () => connect(), { immediate: true });

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
      const moreParams = { ...params.value, from: from.toISOString(), to: to.toISOString(), fill: "1" };
      const logs = await (
        await fetch(`${loadMoreUrl.value}?${new URLSearchParams(moreParams).toString()}`, { signal })
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

  return { messages, loadOlderLogs, isLoadingMore };
}
