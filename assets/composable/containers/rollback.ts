import type { Container, RollbackTarget } from "@/models/Container";
import config, { withBase } from "@/stores/config";

// The container whose rollback waits for confirmation. One dialog serves every
// surface that offers a rollback: the toolbar and the updates drawer.
const request = shallowRef<Container>();

export const useRollbackRequest = () => request;

export function requestRollback(container: Container) {
  request.value = container;
}

type Keyed = Pick<Container, "host" | "id">;
const key = (c: Keyed) => `${c.host}/${c.id}`;

// Where each container's rollback goes, as the server works it out from the
// host's update events and the container's labels. null is a container with
// none. A local host and an agent host answer alike, and an update Watchtower
// or compose made counts as much as Dozzle's own.
const targets = reactive(new Map<string, RollbackTarget | null>());
const inflight = new Set<string>();

// An answer of "none" is asked again after this, in case the host recorded the
// update that created the container only after the first ask.
const NONE_TTL_MS = 60_000;
// Containers come and go, and nothing else drops these entries.
const MAX_TRACKED = 500;

/** The container's rollback target, once loadRollbackTarget has an answer. */
export function rollbackTargetOf(container: Keyed): RollbackTarget | undefined {
  return targets.get(key(container)) ?? undefined;
}

/**
 * Asks the server for the container's rollback target, once per container id. A
 * container gets a new id with every update or rollback, so the answer never goes
 * stale under it.
 */
export async function loadRollbackTarget(container: Pick<Container, "host" | "id" | "isSwarm" | "state">) {
  if (!config.enableActions || container.isSwarm || container.state === "deleted") return;
  const k = key(container);
  if (targets.has(k) || inflight.has(k)) return;
  inflight.add(k);
  try {
    const response = await fetch(withBase(`/api/hosts/${container.host}/containers/${container.id}/rollback-target`));
    const target = response.status === 200 ? ((await response.json()) as RollbackTarget) : null;
    targets.set(k, target?.imageId ? target : null);
    if (targets.size > MAX_TRACKED) targets.delete(targets.keys().next().value!);
    if (!target) setTimeout(() => targets.get(k) === null && targets.delete(k), NONE_TTL_MS);
  } catch {
    // Left unanswered, so the next look asks again.
  } finally {
    inflight.delete(k);
  }
}

// Containers with a rollback running, so every surface that offers one holds
// off a second while the swap is under way.
const running = reactive(new Set<string>());

export function isRollingBack(container: Keyed) {
  return running.has(key(container));
}

export function markRollingBack(container: Keyed, on: boolean) {
  if (on) running.add(key(container));
  else running.delete(key(container));
}

/**
 * Whether the container keeps data a newer image may have migrated: a writable
 * volume or bind mount. The engine socket is a mount too, but not data.
 */
export function holdsData(container: Pick<Container, "mounts">) {
  return container.mounts.some(
    (m) => m.rw && (m.type === "volume" || m.type === "bind") && !m.source.endsWith(".sock"),
  );
}

/** Whether compose manages the container, so its next pull re-applies the newer image. */
export function composeManaged(container: Pick<Container, "labels">) {
  return Boolean(container.labels["com.docker.compose.project"]);
}
