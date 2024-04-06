type Host = {
  name: string;
  id: string;
  available: boolean;
};
const hosts = computed(() =>
  config.hosts.reduce(
    (acc, item) => {
      acc[item.id] = { ...item, available: true };
      return acc;
    },
    {} as Record<string, Host>,
  ),
);

const markHostAvailable = (id: string, available: boolean) => {
  hosts.value[id].available = available;
};

export function useHosts() {
  return {
    hosts,
    markHostAvailable,
  };
}
