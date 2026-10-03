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
  RangeEdgeLogEntry,
} from "@/models/LogEntry";
import { timeRangeRoute } from "./timeRange";
import { Service, Stack } from "@/models/Stack";
import { Container, GroupedContainers } from "@/models/Container";
import { parseMessage } from "./loadBetween";
import { useLogLoader } from "./logLoader";
import { appendBatch, newerThanOnScreen, newestOnScreen, notOnScreen } from "./logWindow";
import { parseEventData } from "@/utils/events";
import { showAllContainers } from "@/stores/settings";

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
  return useLogStream(computed(() => `/api/groups/${encodeURIComponent(group.value.name)}/logs/stream`));
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

// The Kubernetes tab follows "Show all containers", so a finished Job stays listed.
// Its stream has to follow the same toggle or the Job opens to an empty view.
function k8sLabelsUrl(labels: string) {
  return `/api/labels/${labels}/logs/stream${showAllContainers.value ? "?all=1" : ""}`;
}

export function useNamespaceStream(namespace: Ref<{ name: string }>): LogStreamSource {
  return useLogStream(computed(() => k8sLabelsUrl(`@k8s.namespace:${namespace.value.name}`)));
}

export function useOwnerStream(owner: Ref<{ label: string }>): LogStreamSource {
  return useLogStream(computed(() => k8sLabelsUrl(`${owner.value.label}:true`)));
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
  const { streamConfig, hasComplexLogs, levels, loadingMore, containers, timeRange } = useLoggingContext();
  const router = useRouter();
  // "Live, from 15 minutes ago": the tail starts there and loading older stops there.
  // Only a single container's view carries one.
  const floor = computed(() => (container && timeRange.value.kind === "since" ? timeRange.value.since : undefined));
  let initial = true;
  // Set while a reconnected stream replays lines the view already shows.
  let resuming: ((entry: LogEntry<LogMessage>) => boolean) | null = null;
  let sortNext = false;

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
    {
      floor,
      startEdge: () => {
        const start = floor.value!;
        const earlier = (ms: number) => () =>
          router.replace(timeRangeRoute(container!.value.id, { kind: "since", since: new Date(start.getTime() - ms) }));
        return new RangeEdgeLogEntry(start, "start", [
          { label: "5m", run: earlier(5 * 60_000) },
          { label: "15m", run: earlier(15 * 60_000) },
          { label: "1h", run: earlier(60 * 60_000) },
        ]);
      },
    },
  );

  function flushNow() {
    // Only merged views need this. A single container's stream already arrives in
    // order, and sorting it would reorder stdout against stderr, which are separate
    // pipes the daemon can stamp out of delivery order. With several containers a
    // batch is genuinely interleaved, and sorting every one of them (not just the
    // first) is what lets the opening window be short.
    // A resumed search also lands here once, with matches from the gap mixed into the live tail.
    if (initial || sortNext || allContainers.value.length > 1) {
      buffer.sort((a, b) => a.date.getTime() - b.date.getTime());
    }
    sortNext = false;
    const batch = buffer;
    buffer = [];

    // The first batch that fits opens the view, headed by the load-more row, and
    // is the only one that triggers an immediate alert pass; after that the poll
    // owns the cadence, so log volume cannot drive request volume.
    // An opening burst over maxLogs is still the opening: it has to end `initial`
    // too, or the next small batch lands here and replaces the window it kept.
    const wasInitial = initial;
    initial = false;
    const overflows = messages.value.length + batch.length > config.maxLogs;
    if (wasInitial && !overflows) {
      const head =
        container || containers.value.length > 0 ? [new LoadMoreLogEntry(new Date(), loadOlderLogs)] : messages.value;
      messages.value = [...head, ...batch];
    } else {
      messages.value = appendBatch(messages.value, batch, {
        maxLogs: config.maxLogs,
        paused: scrollingPaused.value === true,
        loadSkipped: loadSkippedLogs,
      });
    }
    if (wasInitial) decorateWithAlerts();
  }

  const flushBuffer = useAdaptiveFlush(flushNow, () => initial);
  let es: EventSource | null = null;
  const reconnect = useSseReconnect({
    connect: () => connect({ clear: false }),
    source: () => es,
    onClosed: checkSession,
  });

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

  const urlWithParams = computed(() => {
    const query = new URLSearchParams(params.value);
    if (floor.value) query.set("since", floor.value.toISOString());
    return withBase(`${url.value}${url.value.includes("?") ? "&" : "?"}${query.toString()}`);
  });

  // Every connect replays each container's tail. A fresh one (first open, or the url
  // changed) starts the view over. A reconnect keeps what is on screen and drops the
  // replayed lines it already has, so a dropped stream is a hiccup rather than a view
  // that blanks and refills.
  function markResuming() {
    flushBuffer.flush();
    // A view fed only by backfill never flushed, and would still treat the next batch
    // as the opening one that replaces the window.
    initial = false;
    resuming = newerThanOnScreen(messages.value);
  }

  function connect({ clear } = { clear: true }) {
    close();
    if (clear || (messages.value.length === 0 && buffer.length === 0)) {
      clearMessages();
      resuming = null;
      opened.value = false;
      loading.value = true;
      initial = true;
    } else {
      markResuming();
    }
    error.value = false;
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

      // A stopped container answers every connect with its stop, stamped when it
      // stopped. Before a "since" floor it is not part of the window, and it would
      // hide the empty state that offers the nearest lines. On a reconnect it is the
      // row already on screen.
      if (floor.value && containerEvent.date < floor.value) return;
      const sameEvent = (m: LogEntry<LogMessage>) =>
        m instanceof ContainerEventLogEntry &&
        m.id === containerEvent.id &&
        m.containerID === containerEvent.containerID &&
        m.event === containerEvent.event;
      if (messages.value.some(sameEvent) || buffer.some(sameEvent)) return;

      buffer.push(containerEvent);
      flushBuffer();
      flushBuffer.flush();
    });

    es.addEventListener("logs-backfill", (e) => {
      const data = parseEventData<LogEvent[]>(e);
      let logs = data.map((e) => asLogEntry(e)).filter(notOnScreen(messages.value));
      if (resuming) {
        // A search streams live from the moment it connects, so after a resume the
        // backfill carries the matches written while the stream was down. Those belong
        // at the bottom with the live tail, not above everything.
        // The walk is by time across every container, so the cutoff is the view's
        // newest line rather than each container's.
        const newest = newestOnScreen(messages.value);
        const gap = logs.filter((l) => l.date.getTime() > newest);
        logs = logs.filter((l) => l.date.getTime() <= newest);
        if (gap.length > 0) {
          buffer.push(...gap);
          sortNext = true;
          flushBuffer();
        }
      }
      if (logs.length === 0) return;
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
        const entry = parseMessage(e.data);
        if (resuming && !resuming(entry)) return;
        buffer.push(entry);
        flushBuffer();
      }
    };
    es.onerror = () => {
      error.value = true;
      // The browser may retry on its own (CONNECTING), and that stream replays the tail
      // too. When it has given up (CLOSED), useSseReconnect opens a new one.
      if (messages.value.length > 0 || buffer.length > 0) markResuming();
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
