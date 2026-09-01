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

// Dozzle's own container can only be updated in place when it is a swarm
// service; a standalone container cannot stop itself mid-update.
function isSelfUpdatable(container: Container) {
  return !container.image.includes("amir20/dozzle") || container.isSwarm;
}

export const useImageUpdate = (container: Ref<Container>) => {
  const { t } = useI18n();
  const { showToast } = useToast();
  const { update } = useContainerActions(container);

  const key = computed(() => `${container.value.host}/${container.value.id}`);
  const result = computed(() => results.get(key.value));
  const checking = computed(() => checkingKeys.has(key.value));

  async function check(force = false) {
    if (config.imageCheckMode === "off") return;
    if (!force && config.imageCheckMode !== "automatic") return;

    const currentKey = key.value;
    if (!force && (results.has(currentKey) || inflight.has(currentKey))) return;

    const request = (async () => {
      checkingKeys.add(currentKey);
      try {
        const url = `/api/hosts/${container.value.host}/containers/${container.value.id}/image/check`;
        const response = await fetch(withBase(force ? `${url}?force=true` : url));
        if (!response.ok) return;

        results.set(currentKey, (await response.json()) as ImageUpdateResult);
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
  const updatable = computed(() => config.enableActions && isSelfUpdatable(container.value));

  // Dozzle's own standalone container cannot be updated in place, so the alert
  // points at the release notes instead of a button.
  const isSelf = computed(() => !isSelfUpdatable(container.value));

  function dismiss() {
    if (updateKey.value) {
      dismissedImageUpdates.value = new Set([...dismissedImageUpdates.value, updateKey.value]);
    }
  }

  watch(
    [key, showAlert, showImageUpdateAlert],
    () => {
      if (!showAlert.value || !showImageUpdateAlert.value) return;

      const notifyKey = `${key.value}@${result.value?.remoteDigest}`;
      if (notified.has(notifyKey)) return;
      notified.add(notifyKey);

      showToast({
        id: `image-update-${key.value}`,
        title: t("toolbar.update-available"),
        message: container.value.image,
        type: "info",
        action: updatable.value ? { label: t("toolbar.update"), handler: () => update() } : undefined,
      });
    },
    { immediate: true },
  );

  watch(key, () => check(), { immediate: true });

  return { result, checking, check, updateAvailable, showAlert, updatable, isSelf, dismissed, dismiss };
};
