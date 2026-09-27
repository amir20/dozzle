export type Host = {
  id: string;
  name: string;
  nCPU: number;
  memTotal: number;
  type: "agent" | "local" | "remote" | "swarm" | "k8s";
  endpoint: string;
  available: boolean;
  dockerVersion: string;
  runtime?: "docker" | "podman";
  agentVersion: string;
  group?: string;
  // set when the server re-keyed this host because its agent came back under a
  // new id, so we drop the entry it used to live under instead of listing the
  // same machine twice
  replacesId?: string;
  // set on the one update sent when an agent is removed, so every tab drops it
  removed?: boolean;
};

const hosts = ref(
  config.hosts
    .sort((a, b) => a.name.localeCompare(b.name))
    .reduce(
      (acc, item) => {
        acc[item.id] = item;
        return acc;
      },
      {} as Record<string, Host>,
    ),
);
const updateHost = (host: Host) => {
  delete hosts.value[host.endpoint];
  if (host.replacesId) {
    delete hosts.value[host.replacesId];
  }
  hosts.value[host.id] = host;
  return host;
};

// An agent that never connected is listed under its endpoint, so both keys are
// cleared. Other tabs hear about it as an update-host with removed set.
const removeHost = (...ids: (string | undefined)[]) => {
  for (const id of ids) {
    if (id) delete hosts.value[id];
  }
};

export function useHosts() {
  return {
    hosts,
    updateHost,
    removeHost,
  };
}
