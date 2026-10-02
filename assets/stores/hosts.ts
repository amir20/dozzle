export type Host = {
  id: string;
  name: string;
  nCPU: number;
  memTotal: number;
  metricsAvailable?: boolean;
  load1?: number;
  load5?: number;
  load15?: number;
  uptime?: number;
  diskTotal?: number;
  diskFree?: number;
  // extra drives mounted under /host/disks/<name>; diskTotal/diskFree stay Docker's own
  disks?: { name: string; total: number; free: number }[];
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

export type HostMetrics = Pick<
  Host,
  "id" | "metricsAvailable" | "load1" | "load5" | "load15" | "uptime" | "diskTotal" | "diskFree" | "disks"
>;

// Merges a host-metrics tick into the host already known, leaving available and
// type alone. A tick for a host this tab has not heard of yet is dropped.
const updateHostMetrics = ({ id, ...metrics }: HostMetrics) => {
  const host = hosts.value[id];
  if (host) Object.assign(host, metrics);
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
    updateHostMetrics,
    removeHost,
  };
}
