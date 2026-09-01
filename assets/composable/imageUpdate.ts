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

// Dozzle's own container can only be updated in place when it is a swarm
// service; a standalone container cannot stop itself mid-update.
function isSelfUpdatable(container: Container) {
  return !container.image.includes("amir20/dozzle") || container.isSwarm;
}

export const useImageUpdate = (container: Ref<Container>) => {
  const result = ref<ImageUpdateResult | undefined>();
  const checking = ref(false);

  async function check(force = false) {
    if (config.imageCheckMode === "off") return;
    if (!force && config.imageCheckMode !== "automatic") return;

    checking.value = true;
    try {
      const url = `/api/hosts/${container.value.host}/containers/${container.value.id}/image/check`;
      const response = await fetch(withBase(force ? `${url}?force=true` : url));
      if (!response.ok) return;

      result.value = (await response.json()) as ImageUpdateResult;
    } catch {
      // A failed check is not worth surfacing; the dot simply stays hidden.
    } finally {
      checking.value = false;
    }
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
    () => [container.value.host, container.value.id],
    () => {
      result.value = undefined;
      check();
    },
    { immediate: true },
  );

  return { result, checking, check, updateAvailable, showAlert, updatable, isSelf, dismissed, dismiss };
};
