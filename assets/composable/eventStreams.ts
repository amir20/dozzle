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
import { parseMessage } from "@/composable/loadBetween";
import { useLogLoader } from "@/composable/logLoader";
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
  // A plain array, not a ref: nothing renders the buffer, and rebuilding a new
  // array per arriving line turned the ~100-line opening burst into O(n^2)
  // copies right when the view is trying to paint for the first time.
  let buffer: LogEntry<LogMessage>[] = [];
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
  // The alert layer — dedupe state, the 15s poll, and the merge itself — lives
  // in useAlertMerger, shared with the historical view. All the stream has to
  // do is say when it has assembled a new window.
  const { loadOlderLogs, loadSkippedLogs, decorateWithAlerts } = useLogLoader(
    messages,
    allContainers,
    params,
    loadingMore,
  );

  function flushNow() {
    // Only the first assembly triggers an immediate alert pass; after that the
    // poll owns the cadence, so log volume cannot drive request volume.
    let wasInitial = false;
    // Only merged views need this. A single container's stream already arrives in
    // order, and sorting it would reorder stdout against stderr, which are separate
    // pipes the daemon can stamp out of delivery order. With several containers a
    // batch is genuinely interleaved, and sorting every one of them (not just the
    // first) is what lets the opening window be short.
    if (initial || allContainers.value.length > 1) {
      buffer.sort((a, b) => a.date.getTime() - b.date.getTime());
    }
    if (messages.value.length + buffer.length > config.maxLogs) {
      if (scrollingPaused.value === true) {
        if (messages.value.at(-1) instanceof SkippedLogsEntry) {
          const lastEvent = messages.value.at(-1) as SkippedLogsEntry;
          const lastItem = buffer.at(-1) as LogEntry<string | JSONObject>;
          lastEvent.addSkippedEntries(buffer.length, lastItem);
        } else {
          const firstItem = buffer.at(0) as LogEntry<string | JSONObject>;
          const lastItem = buffer.at(-1) as LogEntry<string | JSONObject>;
          messages.value = [
            ...messages.value,
            new SkippedLogsEntry(new Date(), buffer.length, firstItem, lastItem, loadSkippedLogs),
          ];
        }
        buffer = [];
      } else {
        if (buffer.length > config.maxLogs / 2) {
          messages.value = buffer.slice(-config.maxLogs / 2);
        } else {
          messages.value = [...messages.value, ...buffer].slice(-config.maxLogs);
        }
        buffer = [];
      }
    } else {
      if (initial) {
        wasInitial = true;
        if (container || containers.value.length > 0) {
          const loadMoreItem = new LoadMoreLogEntry(new Date(), loadOlderLogs);
          messages.value = [loadMoreItem];
        }
        initial = false;
      }
      messages.value = [...messages.value, ...buffer];
      buffer = [];
    }
    if (wasInitial) decorateWithAlerts();
  }

  // Two cadences. Steady state batches hard so a chatty container can't drive a
  // render per line. The opening burst gets a much tighter window instead: the
  // skeleton stays up until the first flush, so the 250ms/1000ms pair spent up
  // to a full second showing nothing on a view whose logs had already arrived.
  const initialFlush = debounce(flushNow, 50, { maxWait: 150 });
  const steadyFlush = debounce(flushNow, 250, { maxWait: 1000 });
  const flushBuffer = Object.assign(() => (initial ? initialFlush() : steadyFlush()), {
    cancel: () => {
      initialFlush.cancel();
      steadyFlush.cancel();
    },
    flush: () => {
      initialFlush.flush();
      steadyFlush.flush();
    },
  });
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

      buffer.push(containerEvent);
      flushBuffer();
      flushBuffer.flush();
    });

    es.addEventListener("logs-backfill", (e) => {
      const data = parseEventData<LogEvent[]>(e);
      const logs = data.map((e) => asLogEntry(e));
      messages.value = [...logs, ...messages.value];
      decorateWithAlerts();
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
        buffer.push(parseMessage(e.data));
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
