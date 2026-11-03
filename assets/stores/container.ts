import { acceptHMRUpdate, defineStore } from "pinia";
import { Ref, UnwrapNestedRefs } from "vue";
import type { ContainerHealth, ContainerJson, ContainerStat } from "@/types/Container";
import type { ContainerUpdate } from "@/models/ContainerUpdate";
import type { AutoPruneRun } from "@/composable/setup/setup";
import { Container } from "@/models/Container";
import i18n from "@/modules/i18n";
import { typedEvents } from "@/utils/events";
import { Host, HostMetrics } from "./hosts";
import { sessionHost } from "@/composable/app/storage";

const { showToast, removeToast } = useToast();
const { updateHost, updateHostMetrics, removeHost } = useHosts();
const { markStale } = useStaleUI();
// @ts-ignore
const { t } = i18n.global;

// Matches updateFeedLimit in internal/web/update_feed.go.
const MAX_RECENT_UPDATES = 20;
const updateKey = (u: ContainerUpdate) => `${u.host}/${u.newId}/${u.at}`;

// What /api/events/stream sends, by event name.
type EventsStream = {
  "server-version": { version: string };
  "containers-changed": ContainerJson[];
  // a live stat always names its container; only buffered points drop the id
  "container-stat": ContainerStat & { id: string };
  "container-event": { actorId: string; name: string; time: string };
  "container-updated": ContainerJson;
  "container-update": ContainerUpdate;
  // null when there is nothing to show, which drops a row from before a reconnect.
  "auto-prune": AutoPruneRun | null;
  "update-host": Host;
  "host-metrics": HostMetrics;
  "container-health": { actorId: string; health: ContainerHealth };
};

export const useContainerStore = defineStore("container", () => {
  const containers: Ref<Container[]> = ref([]);

  // The updates Dozzle made lately, newest first, for the megaphone. The server keeps them
  // in memory and replays the recent ones on every connect, so a reconnect sends ones
  // already here; they are keyed on what an update is unique by.
  const recentUpdates = ref<ContainerUpdate[]>([]);
  // The last scheduled clean up, sent only when it is recent and did something.
  const lastAutoPrune = ref<AutoPruneRun>();

  let es: EventSource | null = null;
  // True once the first list has arrived, and never false again: views gate their
  // mount on it, so dropping it on a reconnect tore every open one down.
  const ready = ref(false);
  // Set from open until the list the server sends on connect has landed. That list
  // covers every host, so it is reconciled as the whole world, and what it carries is
  // a replay rather than news, so nothing in it is flagged as new.
  let replaying = false;

  const allContainersById = computed(() =>
    containers.value.reduce(
      (acc, container) => {
        acc[container.id] = container;
        return acc;
      },
      {} as Record<string, Container>,
    ),
  );

  const visibleContainers = computed(() => {
    // A destroyed container lingers as "deleted" until the next list covering its host
    // drops it (see updateContainers), so leave it out of what is shown and counted.
    const filter = showAllContainers.value
      ? (c: Container) => c.state !== "deleted"
      : (c: Container) => c.state === "running";
    return containers.value.filter(filter);
  });

  let errorTimer: ReturnType<typeof setTimeout> | null = null;
  // Set when the stream has been down long enough to tell the user (the same moment
  // the toast below appears), so the logo can show it too.
  const disconnected = ref(false);

  // the server heartbeats every 20s, so three misses means the connection is gone even if
  // the browser still calls it OPEN
  const reconnect = useSseReconnect({
    connect: () => connect(),
    source: () => es,
    staleAfter: 60_000,
    onClosed: checkSession,
  });

  // Counted on open rather than in connect(): the browser reconnects an EventSource on
  // its own too, and those never go through connect().
  let openedBefore = false;
  function connect() {
    es?.close();
    es = new EventSource(withBase("/api/events/stream"));
    const on = typedEvents<EventsStream>(es);
    es.addEventListener("error", (e) => {
      reconnect.onError();
      // the browser retries on its own while CONNECTING; give it a few goes before saying
      // anything, since a single hiccup resolves itself well inside this window
      if (errorTimer === null && es?.readyState !== EventSource.OPEN) {
        errorTimer = setTimeout(() => {
          errorTimer = null;
          disconnected.value = true;
          showToast(
            {
              id: "events-stream",
              message: t("error.events-stream.message"),
              title: t("error.events-stream.title"),
              type: "error",
            },
            { once: true },
          );
        }, 10000);
      }
    });

    es.addEventListener("ping", () => reconnect.onActivity());

    on("server-version", ({ version }) => {
      if (version !== config.version) {
        markStale();
      }
    });

    on("containers-changed", (payload) => {
      updateContainers(payload, replaying);
      replaying = false;
      ready.value = true;
      // the load-time notice below may have fired against a slow first list; the
      // containers are here now, so it no longer describes anything
      removeToast("events-timeout");
    });
    on("container-stat", (stat) => {
      const container = allContainersById.value[stat.id] as unknown as UnwrapNestedRefs<Container>;
      if (container) {
        const { id, ...rest } = stat;
        container.updateStat(rest);
      }
    });
    on("container-event", (event) => {
      const container = allContainersById.value[event.actorId];
      if (container) {
        switch (event.name) {
          case "die":
            container.state = "exited";
            container.finishedAt = new Date(event.time);
            container.health = undefined;
            break;
          case "destroy":
            container.state = "deleted";
            break;
          case "pause":
            container.state = "paused";
            break;
          case "unpause":
            container.state = "running";
            break;
        }
      }
    });

    on("container-updated", (container) => {
      const existing = allContainersById.value[container.id];
      existing?.applyJSON(container, { whole: true });
    });

    on("container-update", addRecentUpdate);

    on("auto-prune", (run) => (lastAutoPrune.value = run ?? undefined));

    on("update-host", (host) => {
      if (host.removed) {
        removeHost(host.id, host.endpoint);
        // The sidebar would stay on a host that no longer exists, with no way back.
        if (sessionHost.value === host.id || sessionHost.value === host.endpoint) sessionHost.value = null;
        containers.value = containers.value.filter((c) => c.host !== host.id);
        return;
      }
      updateHost(host);
    });

    on("host-metrics", updateHostMetrics);

    on("container-health", (event) => {
      const container = allContainersById.value[event.actorId];
      if (container) {
        container.health = event.health;
      }
    });

    es.onopen = () => {
      if (openedBefore) trackUsage("stream.reconnect");
      openedBefore = true;
      reconnect.onOpen();
      disconnected.value = false;
      if (errorTimer !== null) {
        clearTimeout(errorTimer);
        errorTimer = null;
      }
      removeToast("events-stream");
      // EventSource reconnects on its own without going through connect(), and the server
      // replays the full list on every connect. The containers already here are kept and
      // reconciled against that list: emptying the store first unmounted every open log
      // view, and the remount came back with its scroll, older pages and stream gone.
      replaying = true;
    };
  }

  connect();

  (async function () {
    try {
      await until(ready).toBe(true, { timeout: 8000, throwOnTimeout: true });
    } catch (e) {
      showToast(
        {
          id: "events-timeout",
          message: t("error.events-timeout.message"),
          title: t("error.events-timeout.title"),
          type: "error",
        },
        { once: true },
      );
    }
  })();

  const updateContainers = (containersPayload: ContainerJson[], full = false) => {
    const flagNew = ready.value && !full;
    const existingContainers = containersPayload.filter((c) => allContainersById.value[c.id]);
    const newContainers = containersPayload.filter((c) => !allContainersById.value[c.id]);

    existingContainers.forEach((c) => {
      const existing = allContainersById.value[c.id];
      if (flagNew && existing.state !== "running" && c.state === "running") {
        existing.isNew = true;
      }
      // A start reaches here as a re-list, with no event of its own, so this is the
      // only place a restarted container learns when its new run began.
      existing.applyJSON(c);
    });

    const mapped = newContainers.map((c) => {
      const container = Container.fromJSON(c);
      if (flagNew) {
        container.isNew = true;
      }
      return container;
    });

    // `containers-changed` is the authoritative list for every host it names, so a
    // container of a named host that is missing from it is gone and is dropped here.
    // Nothing else ever removed one: `destroy` only marks the container "deleted", and
    // a container that disappears while the tab is asleep raises no event at all. A tab
    // left open on a host with any churn therefore held every container it had ever
    // seen, each keeping its own stats window -- 3.7 KB for one that lived ten seconds,
    // 24.8 KB once it has filled the window. A day of CI on one host ran to hundreds of
    // megabytes that only a reload freed.
    //
    // Scoped by host, never wholesale, except for the payload sent on connect, which is
    // the only one that covers every host. That one also drops hosts it no longer names
    // (gone, unreachable, or emptied while the tab was disconnected). The others are one
    // host's list after a start, a rename, or a stale-host repair, and treating one of
    // those as the whole world would drop every other host.
    const listedHosts = new Set(containersPayload.map((c) => c.host));
    const stillListed = new Set(containersPayload.map((c) => c.id));
    const kept = containers.value.filter((c) => stillListed.has(c.id) || (!full && !listedHosts.has(c.host)));

    containers.value = [...kept, ...mapped];
  };

  const currentContainer = (id: Ref<string>) => computed(() => allContainersById.value[id.value]);

  const containerNames = computed(() =>
    containers.value.reduce(
      (acc, container) => {
        acc[container.id] = container.name;
        return acc;
      },
      {} as Record<string, string>,
    ),
  );

  const findContainerById = (id: string) => allContainersById.value[id];

  const containersByHost = computed(() =>
    containers.value.reduce(
      (acc, container) => {
        if (!acc[container.host]) {
          acc[container.host] = [];
        }
        acc[container.host].push(container);
        return acc;
      },
      {} as Record<string, Container[]>,
    ),
  );

  function addRecentUpdate(update: ContainerUpdate) {
    const key = updateKey(update);
    if (recentUpdates.value.some((u) => updateKey(u) === key)) return;
    recentUpdates.value = [update, ...recentUpdates.value]
      .sort((a, b) => Date.parse(b.at) - Date.parse(a.at))
      .slice(0, MAX_RECENT_UPDATES);
  }

  return {
    containers,
    recentUpdates,
    lastAutoPrune,
    allContainersById,
    containersByHost,
    visibleContainers,
    currentContainer,
    findContainerById,
    containerNames,
    ready,
    disconnected,
  };
});

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useContainerStore, import.meta.hot));
}
