import type { Container } from "@/models/Container";
import type { Host } from "@/stores/hosts";

// The container whose update waits for confirmation. A single update runs at
// once unless there is something to ask: today, whether Dozzle Cloud should
// watch it. The bulk drawer asks the same thing in its footer.
const request = shallowRef<Container>();

export const useUpdateRequest = () => request;

export function requestUpdate(container: Container) {
  request.value = container;
}

/**
 * Whether Dozzle Cloud can be told about an update of the container. The update
 * is recorded by the host that runs it, and only a Docker host this Dozzle
 * talks to directly reports it: an agent keeps its own, a swarm service rolls
 * out without one, and Dozzle's own update replaces the process that would.
 */
export function cloudWatchable(
  container: Pick<Container, "isSwarm">,
  hostType: Host["type"] | undefined,
  self = false,
) {
  return !self && !container.isSwarm && (hostType === "local" || hostType === "remote");
}
