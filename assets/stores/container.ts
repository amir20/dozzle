import { acceptHMRUpdate, defineStore } from "pinia";
import { Ref, UnwrapNestedRefs } from "vue";
import type { ContainerHealth, ContainerJson, ContainerStat } from "@/types/Container";
import { Container } from "@/models/Container";
import i18n from "@/modules/i18n";
import { parseEventData } from "@/utils/events";
import { Host, HostMetrics } from "./hosts";
import { sessionHost } from "@/composable/app/storage";

const { showToast, removeToast } = useToast();
const { updateHost, updateHostMetrics, removeHost } = useHosts();
const { markStale } = useStaleUI();
// @ts-ignore
const { t } = i18n.global;

// An update renames the outgoing container to <name>-dozzle-old-<short id> and keeps it
// until the replacement has stayed up, which takes 10s or more (see
// internal/container/swap/spec.go OldName). Listed, it would show up as a second,
// oddly named container for that whole time.
const SWAP_OLD_NAME = /^(.+)-dozzle-old-[0-9a-f]{12}$/;

function swapOriginalName(name: string): string | undefined {
  return SWAP_OLD_NAME.exec(name)?.[1];
}

export const useContainerStore = defineStore("container", () => {
  const containers: Ref<Container[]> = ref([]);
  // Containers an update has renamed out of the way. They are kept out of `containers`,
  // so nothing lists or counts them, but stay findable by id under their own name: a log
  // view open on one keeps streaming and follows the replacement once it dies. A
  // rollback renames one back, which moves it back into `containers`.
  const swapping: Ref<Container[]> = ref([]);

  let es: EventSource | null = null;
  // True once the first list has arrived, and never false again: views gate their
  // mount on it, so dropping it on a reconnect tore every open one down.
  const ready = ref(false);
  // Set from open until the list the server sends on connect has landed. That list
  // covers every host, so it is reconciled as the whole world, and what it carries is
  // a replay rather than news, so nothing in it is flagged as new.
  let replaying = false;

  const allContainersById = computed(() =>
    [...containers.value, ...swapping.value].reduce(
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
    es.addEventListener("error", (e) => {
      reconnect.onError();
      // the browser retries on its own while CONNECTING; give it a few goes before saying
      // anything, since a single hiccup resolves itself well inside this window
      if (errorTimer === null && es?.readyState !== EventSource.OPEN) {
        errorTimer = setTimeout(() => {
          errorTimer = null;
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

    es.addEventListener("server-version", (e) => {
      const { version } = parseEventData<{ version: string }>(e);
      if (version !== config.version) {
        markStale();
      }
    });

    es.addEventListener("containers-changed", (e) => {
      updateContainers(parseEventData<ContainerJson[]>(e), replaying);
      replaying = false;
      ready.value = true;
      // the load-time notice below may have fired against a slow first list; the
      // containers are here now, so it no longer describes anything
      removeToast("events-timeout");
    });
    es.addEventListener("container-stat", (e) => {
      const stat = parseEventData<ContainerStat>(e);
      const container = allContainersById.value[stat.id] as unknown as UnwrapNestedRefs<Container>;
      if (container) {
        const { id, ...rest } = stat;
        container.updateStat(rest);
      }
    });
    es.addEventListener("container-event", (e) => {
      const event = parseEventData<{ actorId: string; name: string; time: string }>(e);
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

    es.addEventListener("container-updated", (e) => {
      const container = parseEventData<ContainerJson>(e);
      const existing = allContainersById.value[container.id];
      if (existing) {
        if (swapOriginalName(container.name) !== undefined) {
          hideSwapping(existing);
        } else {
          unhideSwapping(existing);
          existing.name = container.name;
        }
        existing.group = container.group;
        existing.state = container.state;
        existing.health = container.health;
        existing.startedAt = new Date(container.startedAt);
        existing.finishedAt = new Date(container.finishedAt);
        if (container.mountStats) {
          existing.updateMountStats(container.mountStats);
        }
      }
    });

    es.addEventListener("update-host", (e) => {
      const host = parseEventData<Host>(e);
      if (host.removed) {
        removeHost(host.id, host.endpoint);
        // The sidebar would stay on a host that no longer exists, with no way back.
        if (sessionHost.value === host.id || sessionHost.value === host.endpoint) sessionHost.value = null;
        containers.value = containers.value.filter((c) => c.host !== host.id);
        swapping.value = swapping.value.filter((c) => c.host !== host.id);
        return;
      }
      updateHost(host);
    });

    es.addEventListener("host-metrics", (e) => updateHostMetrics(parseEventData<HostMetrics>(e)));

    es.addEventListener("container-health", (e) => {
      const event = parseEventData<{ actorId: string; health: ContainerHealth }>(e);
      const container = allContainersById.value[event.actorId];
      if (container) {
        container.health = event.health;
      }
    });

    es.onopen = () => {
      if (openedBefore) trackUsage("stream.reconnect");
      openedBefore = true;
      reconnect.onOpen();
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

  function hideSwapping(container: Container) {
    if (swapping.value.includes(container)) return;
    containers.value = containers.value.filter((c) => c !== container);
    swapping.value = [...swapping.value, container];
  }

  function unhideSwapping(container: Container) {
    if (!swapping.value.includes(container)) return;
    swapping.value = swapping.value.filter((c) => c !== container);
    containers.value = [...containers.value, container];
  }

  const updateContainers = (containersPayload: ContainerJson[], full = false) => {
    const flagNew = ready.value && !full;
    const listed = new Map(containersPayload.map((c) => [c.id, c]));
    const listedHosts = new Set(containersPayload.map((c) => c.host));
    const previouslySwapping = new Set(swapping.value);

    containersPayload.forEach((c) => {
      const existing = allContainersById.value[c.id];
      if (!existing) return;
      const swapped = swapOriginalName(c.name) !== undefined;
      if (!swapped && flagNew && existing.state !== "running" && c.state === "running") {
        existing.isNew = true;
      }
      existing.state = c.state;
      existing.health = c.health;
      // A swap leftover keeps the name it had, so a view open on it still finds its
      // replacement by name.
      if (!swapped) existing.name = c.name;
      existing.group = c.group;
      // A start reaches here as a re-list, with no event of its own, so this is the
      // only place a restarted container learns when its new run began.
      existing.startedAt = new Date(c.startedAt);
      existing.finishedAt = new Date(c.finishedAt);
    });

    const mapped = containersPayload
      .filter((c) => !allContainersById.value[c.id])
      .map((c) => {
        const container = Container.fromJSON(c);
        const original = swapOriginalName(c.name);
        if (original !== undefined) {
          // A tab opened mid-update: hidden straight away, under its own name.
          container.name = original;
        } else if (flagNew) {
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
    const kept = [...containers.value, ...swapping.value].filter(
      (c) => listed.has(c.id) || (!full && !listedHosts.has(c.host)),
    );

    // What the list calls a swap leftover is hidden, a rollback that renamed one back
    // brings it back, and one the list does not mention stays where it was.
    const hidden = (c: Container) => {
      const json = listed.get(c.id);
      return json ? swapOriginalName(json.name) !== undefined : previouslySwapping.has(c);
    };
    const next = [...kept, ...mapped];
    containers.value = next.filter((c) => !hidden(c));
    swapping.value = next.filter(hidden);
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

  return {
    containers,
    allContainersById,
    containersByHost,
    visibleContainers,
    currentContainer,
    findContainerById,
    containerNames,
    ready,
  };
});

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useContainerStore, import.meta.hot));
}
