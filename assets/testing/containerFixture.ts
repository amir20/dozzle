import { Container, type ContainerInit } from "@/models/Container";

// A Container for a spec. Every field has a plain default, so a test names only the
// ones it is about.
export function containerFixture(init: Partial<ContainerInit> = {}): Container {
  return new Container({
    id: "id",
    created: new Date(),
    startedAt: new Date(),
    finishedAt: new Date(),
    image: "image",
    name: "name",
    command: "command",
    host: "localhost",
    state: "running",
    cpuLimit: 0,
    memoryLimit: 0,
    ...init,
  });
}
