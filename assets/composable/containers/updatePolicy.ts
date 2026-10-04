// Which containers the auto-update schedule updates. Every container is one of:
//
//   auto    updated on the schedule
//   manual  checked and shown as "update available", updated by hand
//   off     never checked, never updated
//
// Its dev.dozzle.update label and the instance-wide mode decide it together, the
// same way internal/updatepolicy does on the server. The mode is one setting:
// off (no container), labelled (the ones labelled auto, the default) or all (every
// container not labelled off). Dozzle itself follows the schedule in every mode.
//
// Server mode only: swarm has no Updates page and Kubernetes cannot update at all.

import type { Container } from "@/models/Container";
import type { ImageUpdateResult } from "@/composable/containers/imageUpdate";

export type UpdatePolicy = "auto" | "manual" | "off";
export type UpdateContainersMode = "off" | "labelled" | "all";

export const UPDATE_CONTAINERS_MODES: UpdateContainersMode[] = ["off", "labelled", "all"];
export const DEFAULT_UPDATE_CONTAINERS_MODE: UpdateContainersMode = "labelled";
export const UPDATE_LABEL = "dev.dozzle.update";

// Older labels, still read on the server so nobody has to edit a compose file.
const LEGACY_AUTO_LABEL = "dev.dozzle.auto-update";
const LEGACY_CHECK_LABEL = "dev.dozzle.update-check";

// Mirrors containerUpdatePolicy in internal/web/update_policy.go.
export interface ContainerUpdatePolicy {
  host: string;
  id: string;
  name: string;
  image: string;
  state: string;
  health?: string;
  // What the container's labels choose, if anything.
  label?: UpdatePolicy;
  // Named volumes, for the warning that comes with "all".
  volumes?: string[];
}

export interface UpdatePolicies {
  mode: UpdateContainersMode;
  containers: ContainerUpdatePolicy[];
}

function parsePolicy(value: string | undefined): UpdatePolicy | undefined {
  switch (value?.trim().toLowerCase()) {
    case "auto":
    case "automatic":
    case "true":
    case "on":
    case "yes":
    case "1":
      return "auto";
    case "manual":
      return "manual";
    case "off":
    case "false":
    case "no":
    case "0":
    case "never":
      return "off";
    default:
      return undefined;
  }
}

/** What a container's labels choose, as updatepolicy.FromLabels reads them. */
export function labelPolicy(labels: Record<string, string> | undefined): UpdatePolicy | undefined {
  if (!labels) return undefined;
  const label = parsePolicy(labels[UPDATE_LABEL]);
  if (label) return label;
  if (parsePolicy(labels[LEGACY_CHECK_LABEL]) === "off") return "off";
  switch (parsePolicy(labels[LEGACY_AUTO_LABEL])) {
    case "auto":
      return "auto";
    case "off":
      return "manual";
  }
  return undefined;
}

/** A container's policy under mode, as updatepolicy.Resolve decides it. */
export function resolvePolicy(label: UpdatePolicy | undefined, mode: UpdateContainersMode): UpdatePolicy {
  if (label === "auto" && mode === "off") return "manual";
  if (label) return label;
  return mode === "all" ? "auto" : "manual";
}

/**
 * Whether the schedule would update this container under mode. Like the scheduler,
 * it never updates a stopped container, whatever its label says.
 */
export function willAutoUpdate(entry: Pick<ContainerUpdatePolicy, "label" | "state">, mode: UpdateContainersMode) {
  return entry.state === "running" && resolvePolicy(entry.label, mode) === "auto";
}

/** Containers "all" would reach that keep data in named volumes. */
export function riskyContainers(list: ContainerUpdatePolicy[]): ContainerUpdatePolicy[] {
  return list.filter((c) => !c.label && willAutoUpdate(c, "all") && (c.volumes?.length ?? 0) > 0);
}

/** Whether a container is on the schedule, for lists that only have the container. */
export function autoUpdates(container: Pick<Container, "labels" | "state">, mode: UpdateContainersMode) {
  return willAutoUpdate({ label: labelPolicy(container.labels), state: container.state }, mode);
}

export type ImageStatusKind = "current" | "available" | "off" | "pinned" | "local" | "private" | "unknown" | "pending";

/** What the containers list says about an image check, in a word. */
export function imageStatusKind(result: ImageUpdateResult | undefined): ImageStatusKind {
  switch (result?.status) {
    case undefined:
      return "pending";
    case "up-to-date":
      return "current";
    case "update-available":
      return "available";
    case "pinned":
      return "pinned";
    case "not-checkable":
      return "local";
    case "auth-required":
      return "private";
    case "skipped":
      return "off";
    default:
      return "unknown";
  }
}

const state = ref<UpdatePolicies | null>(null);
let seq = 0;

export function useUpdatePolicies() {
  async function fetchPolicies() {
    if (config.mode !== "server") return state.value;
    const mine = ++seq;
    try {
      const res = await fetch(withBase("/api/updates/policy"));
      if (res.ok && mine === seq) state.value = (await res.json()) as UpdatePolicies;
    } catch {
      // The list stays empty.
    }
    return state.value;
  }

  return { policies: readonly(state), fetchPolicies };
}
