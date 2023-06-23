import { Container } from "@/models/Container";

const sessionHost = useSessionStorage<string | null>("host", null);

if (config.hosts.length === 1 && !sessionHost.value) {
  sessionHost.value = config.hosts[0];
}

function persistentVisibleKeys(container: ComputedRef<Container>) {
  return computed(() => useStorage(stripVersion(container.value.image) + ":" + container.value.command, []));
}

export { sessionHost, persistentVisibleKeys };
