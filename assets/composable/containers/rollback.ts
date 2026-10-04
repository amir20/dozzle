import type { Container } from "@/models/Container";

// The container whose rollback waits for confirmation. One dialog serves every
// surface that offers a rollback: the toolbar and the updates drawer.
const request = shallowRef<Container>();

export const useRollbackRequest = () => request;

export function requestRollback(container: Container) {
  request.value = container;
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
