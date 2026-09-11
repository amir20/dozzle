import type { MaybeRefOrGetter } from "vue";
import type { CloudAlert } from "./cloudAlerts";

/**
 * What fired lately, anywhere on this instance.
 *
 * The scrollback path asks the opposite question (what fired in this window,
 * for these containers) and is answered by `useCloudAlerts`. This one backs the
 * notifications history and the severity dot on a container, neither of which
 * knows a container list up front.
 *
 * What Cloud adds here is memory. Alerts already splice into the live stream and
 * die on refresh, because nothing local remembers them; with a database behind
 * it the same alert is still there tomorrow. So an instance with no cloud link
 * gets an empty list rather than an error, and the surface says what it would
 * hold.
 */

// Shared across every consumer on the page: the notifications history, the
// per-rule counts and the dot all want the same rows.
const alerts = ref<CloudAlert[]>([]);
const loading = ref(false);
const loaded = ref(false);
const failed = ref(false);

let pending: Promise<void> | null = null;

// Module scope rather than per-consumer: every surface below derives from
// whether the instance is linked *right now*, not from what the last fetch
// happened to return.
const { linked } = useCloudSurface();

// Linking and unlinking both happen without a reload — the settings card clears
// the cloud config in place — and the moment either does, the rows this module
// is holding stop describing the instance in front of the reader. So they are
// dropped, and refetched when there is somewhere to ask again.
//
// The refetch also covers boot. `cloudConfig` is fetched asynchronously, so the
// first `fetchRecentAlerts()` of a page load usually runs while the instance
// still looks unlinked and early-returns; without this the history sat empty
// until the bell's poll came round a minute later.
watch(linked, (isLinked) => {
  alerts.value = [];
  loaded.value = false;
  failed.value = false;
  pending = null;
  if (isLinked) pending = load();
});

/**
 * The newest alert this browser has been shown, as nanoseconds.
 *
 * Per profile rather than per instance: the dot answers "is there something I
 * have not looked at", which is a question about the reader, not the server.
 */
const lastSeenTs = useProfileStorage("lastSeenAlertTs", 0);

async function load(limit = 200): Promise<void> {
  loading.value = true;
  failed.value = false;
  try {
    const res = await fetch(withBase(`/api/cloud/alerts/recent?limit=${limit}`));
    if (!res.ok) {
      // 503 is an unlinked instance, which is an empty history and not a
      // failure worth shouting about.
      failed.value = res.status !== 503;
      alerts.value = [];
      return;
    }
    const body = await res.json();
    alerts.value = body.hits ?? [];
  } catch {
    failed.value = true;
    alerts.value = [];
  } finally {
    loading.value = false;
    loaded.value = true;
    pending = null;
  }
}

export function useRecentAlerts() {
  function fetchRecentAlerts(limit?: number) {
    if (!linked.value) {
      alerts.value = [];
      loaded.value = true;
      return Promise.resolve();
    }
    pending ??= load(limit);
    return pending;
  }

  /** Re-reads the list, ignoring the once-per-page cache. What the bell polls. */
  function refreshRecentAlerts(limit?: number) {
    if (!linked.value || pending) return Promise.resolve();
    return load(limit);
  }

  /** Newest first, which is the only order any of these surfaces wants.
   *
   *  Empty while unlinked: an unlinked instance has no memory, so it has no
   *  history — including the rows an earlier link left behind in this tab.
   *  Gating it here rather than in each surface is what keeps a stale row out of
   *  the history, both bells and the dot on a container row in one move. */
  const newestFirst = computed(() => (linked.value ? [...alerts.value].sort((a, b) => b.ts - a.ts) : []));

  /**
   * The most recent alert per container, for the severity dot. A container with
   * three alerts gets one dot, carrying the newest.
   */
  const byContainer = computed(() => {
    const map = new Map<string, CloudAlert>();
    for (const alert of newestFirst.value) {
      if (!map.has(alert.containerId)) map.set(alert.containerId, alert);
    }
    return map;
  });

  /** Alerts one rule raised, for the activity line on its card. */
  function forSubscription(id: string | number) {
    const key = String(id);
    return computed(() => newestFirst.value.filter((a) => a.subscriptionId === key));
  }

  /** Something fired that this reader has not opened the history for. */
  const unseen = computed(() => {
    const newest = newestFirst.value[0];
    return !!newest && newest.ts > lastSeenTs.value;
  });

  /**
   * The same question asked of one set of containers.
   *
   * A dot has to ask whatever its destination answers. The nav's bell opens the
   * notifications page, which shows the instance, so it keeps `unseen`. The
   * rail's bell opens a panel scoped to the view, so an instance-wide dot there
   * promised rows the panel then filtered away, and the reader who followed it
   * landed on "nothing has fired on what you're looking at".
   *
   * An empty scope is a view with no containers of its own (home, settings),
   * where the panel shows the instance and so does the dot.
   */
  function unseenIn(ids: MaybeRefOrGetter<Set<string>>) {
    return computed(() => {
      const scope = toValue(ids);
      if (scope.size === 0) return unseen.value;
      return newestFirst.value.some((a) => scope.has(a.containerId) && a.ts > lastSeenTs.value);
    });
  }

  /** Called when the history is actually on screen, not when it is fetched: the
   *  dot goes out because someone looked, not because a request landed.
   *
   *  `upTo` is the newest alert the surface actually rendered. A scoped panel
   *  that marked the instance's newest seen put out the nav's bell for alerts
   *  the reader never saw, so the watermark moves only as far as the rows that
   *  were on screen. Monotonic either way. */
  function markAlertsSeen(upTo?: CloudAlert) {
    const newest = upTo ?? newestFirst.value[0];
    if (newest && newest.ts > lastSeenTs.value) lastSeenTs.value = newest.ts;
  }

  return {
    alerts: newestFirst,
    byContainer,
    forSubscription,
    fetchRecentAlerts,
    refreshRecentAlerts,
    unseen,
    unseenIn,
    markAlertsSeen,
    loading,
    loaded,
    failed,
  };
}
