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
// session on a busy host would grow them without limit.
const MAX_TRACKED = 200;

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

// Dozzle cannot stop itself to update, so its own container is a special
// case. Only the container this browser is talking to counts: a Dozzle agent
// on a remote host is an ordinary container that updates over RPC like any
// other, and a swarm service is recreated by the orchestrator rather than by
// the process itself.
//
// The image name is matched because the backend does not report which
// container Dozzle runs in. A renamed or mirrored image (my-registry/dozzle)
// is therefore not recognised, and would offer an update button that fails.
function isSelf(container: Container) {
  if (!container.image.includes("amir20/dozzle")) return false;
  if (container.isSwarm) return false;

  return config.hosts.find((host) => host.id === container.host)?.type === "local";
}

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

  // Whether Dozzle can perform the update itself. Independent of whether an
  // update is currently available, so the existing manual pull button stays
  // available exactly as before.
  const updatable = computed(() => config.enableActions && !selfContainer.value);

  // Dozzle's own standalone container cannot be updated in place, so the alert
  // points at the release notes instead of a button.
  const selfContainer = computed(() => isSelf(container.value));

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
      // whether Dozzle is allowed to act and whether it can act on itself.
      // vue-i18n does not escape interpolated values and the notice is
      // rendered as HTML so it can carry a docs link. Image names are
      // arbitrary strings, in k8s especially.
      let message = t("alert.image-update.message", { image: escapeHtml(container.value.image) });
      if (selfContainer.value) {
        message += " " + t("alert.image-update.self");
      } else if (!config.enableActions) {
        message += " " + t("alert.image-update.enable-actions");
      }

      showToast({
        id: `image-update-${key.value}`,
        title: t("alert.image-update.title"),
        message,
        type: "info",
        action: updatable.value ? { label: t("toolbar.update"), handler: () => update() } : undefined,
        secondaryAction: { label: t("toolbar.dismiss-update"), handler: () => dismiss() },
      });
    },
    { immediate: true },
  );

  watch(key, () => check(), { immediate: true });

  return { result, checking, check, updateAvailable, showAlert, updatable, isSelf: selfContainer, dismissed, dismiss };
};
