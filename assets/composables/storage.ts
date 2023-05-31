import { Container } from "@/models/Container";

const sessionHost = useSessionStorage("host", "localhost");

function persistentVisibleKeys(container: ComputedRef<Container>) {
  return computed(() => useStorage(stripVersion(container.value.image) + ":" + container.value.command, []));
}

export { sessionHost, persistentVisibleKeys };
