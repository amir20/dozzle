import { ShallowRef, type Ref } from "vue";

import debounce from "lodash.debounce";
import {
  type LogEvent,
  type LogMessage,
  LogEntry,
  asLogEntry,
  ContainerEventLogEntry,
  ComplexLogEntry,
  LoadMoreLogEntry,
} from "@/models/LogEntry";
import { Service, Stack } from "@/models/Stack";
import { Container, GroupedContainers } from "@/models/Container";
import { parseMessage } from "./loadBetween";
import { useLogLoader } from "./logLoader";
import { appendBatch } from "./logWindow";
import { parseEventData } from "@/utils/events";

const { isSearching, appliedSearchFilter, inverseFilter } = useSearchFilter();

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

function useLabelStream(labels: () => string): LogStreamSource {
  return useLogStream(computed(() => `/api/labels/${labels()}/logs/stream`));
}

export function useStackStream(stack: Ref<Stack>): LogStreamSource {
  return useLabelStream(() => `com.docker.stack.namespace:${stack.value.name}`);
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
  return useLabelStream(() => `com.docker.swarm.service.name:${service.value.name}`);
}

export function useNamespaceStream(namespace: Ref<{ name: string }>): LogStreamSource {
  return useLabelStream(() => `@k8s.namespace:${namespace.value.name}`);
}

export function useOwnerStream(owner: Ref<{ label: string }>): LogStreamSource {
  return useLabelStream(() => `${owner.value.label}:true`);
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
      params.append("filter", appliedSearchFilter.value);
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
    // Only merged views need this. A single container's stream already arrives in
    // order, and sorting it would reorder stdout against stderr, which are separate
    // pipes the daemon can stamp out of delivery order. With several containers a
    // batch is genuinely interleaved, and sorting every one of them (not just the
    // first) is what lets the opening window be short.
    if (initial || allContainers.value.length > 1) {
      buffer.sort((a, b) => a.date.getTime() - b.date.getTime());
    }
    const batch = buffer;
    buffer = [];

    // The first batch that fits opens the view, headed by the load-more row, and
    // is the only one that triggers an immediate alert pass; after that the poll
    // owns the cadence, so log volume cannot drive request volume.
    const overflows = messages.value.length + batch.length > config.maxLogs;
    if (initial && !overflows) {
      initial = false;
      const head =
        container || containers.value.length > 0 ? [new LoadMoreLogEntry(new Date(), loadOlderLogs)] : messages.value;
      messages.value = [...head, ...batch];
      decorateWithAlerts();
      return;
    }

    messages.value = appendBatch(messages.value, batch, {
      maxLogs: config.maxLogs,
      paused: scrollingPaused.value === true,
      loadSkipped: loadSkippedLogs,
    });
  }

  const flushBuffer = useAdaptiveFlush(flushNow, () => initial);
  let es: EventSource | null = null;
  const reconnect = useSseReconnect({ connect: () => connect({ clear: true }), source: () => es });

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
      const data = parseEventData<Omit<SearchStatus, "active">>(e);
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
      // CLOSED means the browser has stopped retrying, so the log view would sit empty
      // until a manual reload. Reconnecting drops and refetches rather than resuming,
      // since the backfill the server replays would otherwise duplicate what is on screen.
      reconnect.onError();
    };
    es.onopen = () => {
      reconnect.onOpen();
      loading.value = false;
      opened.value = true;
      error.value = false;
    };
  }

  watch(urlWithParams, () => connect(), { immediate: true });

  onScopeDispose(() => {
    reconnect.dispose();
    close();
  });

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

// Two cadences. Steady state batches hard so a chatty container can't drive a
// render per line. The opening burst gets a much tighter window instead: the
// skeleton stays up until the first flush, so the 250ms/1000ms pair spent up
// to a full second showing nothing on a view whose logs had already arrived.
function useAdaptiveFlush(fn: () => void, isInitial: () => boolean) {
  const initialFlush = debounce(fn, 50, { maxWait: 150 });
  const steadyFlush = debounce(fn, 250, { maxWait: 1000 });
  return Object.assign(() => (isInitial() ? initialFlush() : steadyFlush()), {
    cancel: () => {
      initialFlush.cancel();
      steadyFlush.cancel();
    },
    flush: () => {
      initialFlush.flush();
      steadyFlush.flush();
    },
  });
}
