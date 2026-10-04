import { Container } from "@/models/Container";

export type ImageUpdateStatus =
  "up-to-date" | "update-available" | "pinned" | "not-checkable" | "auth-required" | "skipped" | "unknown";

export interface ImageUpdateResult {
  status: ImageUpdateStatus;
  image: string;
  localDigest?: string;
  remoteDigest?: string;
  checkedAt: string;
  reason?: string;
}

// Results are shared across every consumer of a container so the toolbar and
// the notification agree, and so a container is only fetched once no matter
// how many components ask about it.
const results = reactive(new Map<string, ImageUpdateResult>());
const checkingKeys = reactive(new Set<string>());
const inflight = new Map<string, Promise<void>>();

// Guards the notification so navigating back to a container does not re-fire
// it for an update the user has already seen this session.
const notified = new Set<string>();

// Containers come and go, and nothing else removes these entries, so a long
// session on a busy host would grow them without limit. Sized so a
// dashboard-wide check on a big install still fits in one pass.
const MAX_TRACKED = 1000;

function trim(collection: Map<string, unknown> | Set<string>) {
  const excess = collection.size - MAX_TRACKED;
  if (excess <= 0) return;

  // Insertion order, so this drops the least recently added keys first.
  for (const key of [...collection.keys()].slice(0, excess)) {
    collection.delete(key);
  }
}

// A result older than this is re-checked when the container is viewed again,
// so a tab left open for days does not keep reporting a stale answer.
const STALE_AFTER = 30 * 60 * 1000;

// Dozzle updating itself goes away mid-update, so its own container is a special
// case: the page waits for the replacement instead of reporting success.
// Only the container this browser is talking to counts: a Dozzle agent
// on a remote host is an ordinary container that updates over RPC like any
// other, and a swarm service is recreated by the orchestrator rather than by
// the process itself.
//
// Matched by container id, the same way the backend decides to self-update, so
// a renamed image still counts and a second Dozzle on the host does not.
export function isSelf(container: Container) {
  const self = config.selfContainerId;
  if (!self || container.id.length < 12 || !self.startsWith(container.id)) return false;
  if (container.isSwarm) return false;

  return config.hosts.find((host) => host.id === container.host)?.type === "local";
}

// Without selfContainerId (Podman, for one) the backend cannot tell its own
// container apart and refuses to update anything that looks like Dozzle, so the
// button is not offered there.
function mayBeSelf(container: Container) {
  if (config.selfContainerId || container.isSwarm) return false;
  if (!container.image.includes("amir20/dozzle")) return false;
  return config.hosts.find((host) => host.id === container.host)?.type === "local";
}

// Same rule as the backend's Updatable: a stopped standalone container is still
// checked, so it shows that an update is waiting, but an exited swarm task is
// history the orchestrator already replaced.
function offerable(container: Container) {
  if (container.state === "deleted") return false;
  return !container.isSwarm || container.state === "running";
}

/**
 * Whether Dozzle can update this container itself, whether or not an update is
 * waiting. Kubernetes rolls out images through the workload, so a single pod has
 * nothing to update, and a stopped container is never updated: it has to be
 * started first. Shared by the toolbar and the command palette.
 */
export function canUpdate(container: Container) {
  return (
    !!config.enableActions &&
    config.mode !== "k8s" &&
    !mayBeSelf(container) &&
    offerable(container) &&
    container.state === "running"
  );
}

const checkingAll = ref(false);
let checkedAllAt = 0;

// Checks every container the dashboard can see in one request, filling the
// same results the per-container menu reads, so both always agree.
async function checkAll(force = false) {
  if (config.imageCheckMode === "off") return;
  if (!force && config.imageCheckMode !== "automatic") return;
  if (checkingAll.value) return;
  if (!force && Date.now() - checkedAllAt < STALE_AFTER) return;

  checkingAll.value = true;
  try {
    const response = await fetch(withBase(force ? "/api/image/check?force=true" : "/api/image/check"));
    if (!response.ok) return;
    const checks = (await response.json()) as { host: string; id: string; result: ImageUpdateResult }[];
    for (const { host, id, result } of checks) {
      results.set(`${host}/${id}`, result);
    }
    trim(results);
    checkedAllAt = Date.now();
  } catch {
    // Same as a single check: a failure leaves the dashboard quiet.
  } finally {
    checkingAll.value = false;
  }
}

// What the dashboard lists as having an update: containers with a newer image
// that Dozzle is able to update itself once they run.
export const useImageUpdates = () => {
  const hasUpdate = (container: Container) =>
    offerable(container) &&
    results.get(`${container.host}/${container.id}`)?.status === "update-available" &&
    !mayBeSelf(container);

  // The last check of a container, for lists that show every container's status.
  const resultFor = (container: Pick<Container, "host" | "id">) => results.get(`${container.host}/${container.id}`);

  return { checkAll, checking: readonly(checkingAll), hasUpdate, isSelf, resultFor };
};

export const useImageUpdate = (container: Ref<Container>, historical: Ref<boolean> | boolean = false) => {
  const { t } = useI18n();
  const { showToast, removeToast } = useToast();
  const { update } = useContainerActions(container);

  const isHistorical = computed(() => unref(historical));

  const key = computed(() => `${container.value.host}/${container.value.id}`);
  const result = computed(() => results.get(key.value));
  const checking = computed(() => checkingKeys.has(key.value));

  async function check(force = false) {
    if (config.imageCheckMode === "off") return;
    if (!force && config.imageCheckMode !== "automatic") return;
    // Historical logs describe a container that is gone; there is nothing to
    // update and nothing worth telling anyone about.
    if (isHistorical.value) return;

    const currentKey = key.value;
    const known = results.get(currentKey);
    const fresh = known && Date.now() - new Date(known.checkedAt).getTime() < STALE_AFTER;
    if (!force && (fresh || inflight.has(currentKey))) return;

    const request = (async () => {
      checkingKeys.add(currentKey);
      try {
        const url = `/api/hosts/${container.value.host}/containers/${container.value.id}/image/check`;
        const response = await fetch(withBase(force ? `${url}?force=true` : url));
        if (!response.ok) return;

        results.set(currentKey, (await response.json()) as ImageUpdateResult);
        trim(results);
      } catch {
        // A failed check is not worth surfacing; the menu simply stays quiet.
      } finally {
        checkingKeys.delete(currentKey);
        inflight.delete(currentKey);
      }
    })();

    inflight.set(currentKey, request);
    return request;
  }

  // Identifies this specific update so dismissing it stays dismissed until the
  // image actually moves again. Without this, a :latest container would alert
  // on every visit.
  const updateKey = computed(() =>
    result.value?.remoteDigest ? `${result.value.image}@${result.value.remoteDigest}` : undefined,
  );

  const updateAvailable = computed(() => result.value?.status === "update-available");

  const dismissed = computed(() => !!updateKey.value && dismissedImageUpdates.value.has(updateKey.value));

  // The alert is informational, so it shows whether or not actions are on.
  const showAlert = computed(() => updateAvailable.value && !dismissed.value);

  const updatable = computed(() => canUpdate(container.value));

  // Dozzle's own standalone container replaces itself through a helper, so the
  // update waits for the new process and the menu also links the release notes.
  const selfContainer = computed(() => isSelf(container.value));

  const runUpdate = () => update({ self: selfContainer.value });

  function dismiss() {
    if (updateKey.value) {
      dismissedImageUpdates.value = new Set([...dismissedImageUpdates.value, updateKey.value]);
    }
    // Dismissing from the menu has to clear a notice that is already on
    // screen, otherwise it lingers with no way to act on it.
    removeToast(`image-update-${key.value}`);
  }

  watch(
    [key, showAlert, showImageUpdateAlert],
    () => {
      if (!showAlert.value || !showImageUpdateAlert.value || isHistorical.value) return;

      const notifyKey = `${key.value}@${result.value?.remoteDigest}`;
      if (notified.has(notifyKey)) return;
      notified.add(notifyKey);
      trim(notified);

      // The message explains what can be done about it, which differs by
      // whether Dozzle is allowed to act.
      // vue-i18n does not escape interpolated values and the notice is
      // rendered as HTML so it can carry a docs link. Image names are
      // arbitrary strings, in k8s especially.
      let message = t("alert.image-update.message", { image: escapeHtml(container.value.image) });
      // Kubernetes rolls images out through the workload, so turning actions
      // on would not give this pod an update button.
      if (!config.enableActions && config.mode !== "k8s") {
        message += " " + t("alert.image-update.enable-actions");
      }

      showToast({
        id: `image-update-${key.value}`,
        title: t("alert.image-update.title"),
        message,
        type: "info",
        action: updatable.value ? { label: t("toolbar.update"), handler: () => runUpdate() } : undefined,
        secondaryAction: { label: t("toolbar.dismiss-update"), handler: () => dismiss() },
      });
    },
    { immediate: true },
  );

  watch(key, () => check(), { immediate: true });

  return {
    result,
    checking,
    check,
    updateAvailable,
    showAlert,
    updatable,
    isSelf: selfContainer,
    update: runUpdate,
    dismissed,
    dismiss,
  };
};
