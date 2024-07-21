export type Host = {
  id: string;
  name: string;
  nCPU: number;
  memTotal: number;
  type: string;
  endpoint: string;
  available: boolean;
};

const hosts = ref(
  config.hosts.reduce(
    (acc, item) => {
      acc[item.id] = item;
      return acc;
    },
    {} as Record<string, Host>,
  ),
);
const updateHost = (host: Host) => {
  delete hosts.value[host.endpoint];
  hosts.value[host.id] = host;
  return host;
};

export function useHosts() {
  return {
    hosts,
    updateHost,
  };
}
