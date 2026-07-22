import { ShallowRef, type Ref } from "vue";

import debounce from "lodash.debounce";
import {
  type LogEvent,
  type JSONObject,
  type LogMessage,
  LogEntry,
  asLogEntry,
  ContainerEventLogEntry,
  ComplexLogEntry,
  SkippedLogsEntry,
  LoadMoreLogEntry,
} from "@/models/LogEntry";
import { Service, Stack } from "@/models/Stack";
import { Container, GroupedContainers } from "@/models/Container";
import { parseMessage, loadBetween } from "@/composable/loadBetween";
import { useLogLoader } from "@/composable/logLoader";
import { loggingContextKey } from "@/composable/logContext";
import { parseEventData } from "@/utils/events";

const { isSearching, debouncedSearchFilter, inverseFilter } = useSearchFilter();

export function useContainerStream(container: Ref<Container>): LogStreamSource {
  const url = computed(() => `/api/hosts/${container.value.host}/containers/${container.value.id}/logs/stream`);
  return useLogStream(url, container);
}

export function useHostStream(host: Ref<Host>): LogStreamSource {
  return useLogStream(computed(() => `/api/hosts/${host.value.id}/logs/stream`));
}

export function useHostGroupStream(group: Ref<{ name: string }>): LogStreamSource {
  return useLogStream(computed(() => `/api/host-groups/${encodeURIComponent(group.value.name)}/logs/stream`));
}

export function useStackStream(stack: Ref<Stack>): LogStreamSource {
  const labels = computed(() => `com.docker.stack.namespace:${stack.value.name}`);
  return useLogStream(computed(() => `/api/labels/${labels.value}/logs/stream`));
}

export function useGroupedStream(group: Ref<GroupedContainers>): LogStreamSource {
  return useLogStream(computed(() => `/api/groups/${group.value.name}/logs/stream`));
}

export function useMergedStream(containers: Ref<Container[]>): LogStreamSource {
  const url = computed(() => {
    const ids = containers.value.map((c) => c.id).join(",");
    return `/api/hosts/${containers.value[0].host}/logs/mergedStream/${ids}`;
  });

  return useLogStream(url);
}

export function useServiceStream(service: Ref<Service>): LogStreamSource {
  const labels = computed(() => `com.docker.swarm.service.name:${service.value.name}`);
  return useLogStream(computed(() => `/api/labels/${labels.value}/logs/stream`));
}

export function useNamespaceStream(namespace: Ref<{ name: string }>): LogStreamSource {
  const labels = computed(() => `@k8s.namespace:${namespace.value.name}`);
  return useLogStream(computed(() => `/api/labels/${labels.value}/logs/stream`));
}

export function useOwnerStream(owner: Ref<{ label: string }>): LogStreamSource {
  const labels = computed(() => `${owner.value.label}:true`);
  return useLogStream(computed(() => `/api/labels/${labels.value}/logs/stream`));
}

export type SearchStatus = {
  active: boolean;
  done: boolean;
  matches: number;
  scannedTo?: string;
  reason?: "capped" | "exhausted";
};

export type LogStreamSource = ReturnType<typeof useLogStream>;

function useLogStream(url: Ref<string>, container?: Ref<Container>) {
  const messages: ShallowRef<LogEntry<LogMessage>[]> = shallowRef([]);
  const buffer: ShallowRef<LogEntry<LogMessage>[]> = shallowRef([]);
  const opened = ref(false);
  const loading = ref(true);
  const error = ref(false);
  const searchStatus = ref<SearchStatus>({ active: false, done: false, matches: 0 });
  const { paused: scrollingPaused } = useScrollContext();
  const { streamConfig, hasComplexLogs, levels, loadingMore, containers } = useLoggingContext();
  let initial = true;

  const params = computed(() => {
    const params = new URLSearchParams();
    if (streamConfig.value.stdout) params.append("stdout", "1");
    if (streamConfig.value.stderr) params.append("stderr", "1");
    if (isSearching.value) {
      params.append("filter", debouncedSearchFilter.value);
      if (inverseFilter.value) params.append("inverse", "true");
    }
    for (const level of levels.value) {
      params.append("levels", level);
    }
    return params;
  });

  const allContainers = computed(() => (container ? [container.value] : containers.value));
  const { loadOlderLogs, loadSkippedLogs } = useLogLoader(messages, allContainers, params, loadingMore);

  // Jump straight to the container's first lines by fetching only the oldest
  // window (from each container's creation), instead of lazily loading every
  // line in between. Pausing means any live logs that arrive while viewing the
  // head collect as a skipped marker rather than appending; "go to bottom"
  // reconnects to the live tail.
  async function loadOldest() {
    const cs = allContainers.value;
    if (cs.length === 0) return;
    loadingMore.value = true;
    try {
      const perContainer = Math.max(1, Math.ceil(config.maxLogs / cs.length));
      const results = await Promise.all(
        cs.map((c) => loadBetween(c, params, c.created, new Date(), { maxStart: perContainer })),
      );
      const head = results
        .filter(({ signal }) => !signal.aborted)
        .flatMap(({ logs }) => logs)
        .sort((a, b) => a.date.getTime() - b.date.getTime())
        .slice(0, config.maxLogs);
      if (head.length === 0) return;
      scrollingPaused.value = true;
      messages.value = [new LoadMoreLogEntry(new Date(), loadOlderLogs), ...head];
    } catch (err) {
      console.error(err);
    } finally {
      loadingMore.value = false;
    }
  }

  const loggingContextRaw = inject(loggingContextKey);
  if (loggingContextRaw) {
    loggingContextRaw.jumpToOldest = loadOldest;
    loggingContextRaw.reconnect = () => connect();
  }

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
            new SkippedLogsEntry(new Date(), buffer.value.length, firstItem, lastItem, loadSkippedLogs),
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
      if (initial) {
        // sort the buffer the very first time because of multiple logs in parallel
        buffer.value.sort((a, b) => a.date.getTime() - b.date.getTime());

        if (container || containers.value.length > 0) {
          const loadMoreItem = new LoadMoreLogEntry(new Date(), loadOlderLogs);
          messages.value = [loadMoreItem];
        }
        initial = false;
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

  const urlWithParams = computed(() => withBase(`${url.value}?${params.value.toString()}`));

  function connect({ clear } = { clear: true }) {
    close();
    if (clear) clearMessages();
    opened.value = false;
    loading.value = true;
    error.value = false;
    initial = true;
    searchStatus.value = { active: isSearching.value, done: false, matches: 0 };
    es = new EventSource(urlWithParams.value);
    es.addEventListener("container-event", (e) => {
      const event = parseEventData<{
        actorId: string;
        name: "container-stopped" | "container-started";
        time: string;
      }>(e);
      const containerEvent = new ContainerEventLogEntry(
        event.name == "container-started" ? "Container started" : "Container stopped",
        event.actorId,
        new Date(event.time),
        event.name,
      );

      buffer.value = [...buffer.value, containerEvent];
      flushBuffer();
      flushBuffer.flush();
    });

    es.addEventListener("logs-backfill", (e) => {
      const data = parseEventData<LogEvent[]>(e);
      const logs = data.map((e) => asLogEntry(e));
      messages.value = [...logs, ...messages.value];
    });

    es.addEventListener("search-status", (e) => {
      const data = parseEventData<{
        scannedTo: string;
        matches: number;
        done: boolean;
        reason?: "capped" | "exhausted";
      }>(e);
      searchStatus.value = {
        active: !data.done,
        done: data.done,
        matches: data.matches,
        scannedTo: data.scannedTo,
        reason: data.reason,
      };
    });

    es.onmessage = (e) => {
      if (e.data) {
        buffer.value = [...buffer.value, parseMessage(e.data)];
        flushBuffer();
      }
    };
    es.onerror = () => {
      error.value = true;
    };
    es.onopen = () => {
      loading.value = false;
      opened.value = true;
      error.value = false;
    };
  }

  watch(urlWithParams, () => connect(), { immediate: true });

  onScopeDispose(() => close());

  watch(messages, () => {
    if (messages.value.length > 1) {
      hasComplexLogs.value = messages.value.some((m) => m instanceof ComplexLogEntry);
    }
  });

  return {
    messages,
    opened,
    error,
    loading,
    searchStatus,
  };
}
