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

export function useHosts() {
  return {
    hosts,
    updateHost,
  };
}
