import { beforeEach, describe, expect, test, vi } from "vitest";
import { createPinia, defineStore, setActivePinia } from "pinia";
import { ref } from "vue";

import type { Container } from "@/models/Container";
import type { ContainerState } from "@/types/Container";
import { containerFixture } from "@/testing/containerFixture";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [{ name: "localhost", id: "localhost" }] },
  withBase: (path: string) => path,
}));

const containers = ref<Container[]>([]);

vi.mock("@/stores/container", () => ({
  useContainerStore: defineStore("container", () => ({ containers })),
}));

const { showAllContainers } = await import("@/stores/settings");
const { useSwarmStore } = await import("./swarm");

function task(id: string, state: ContainerState, labels: Record<string, string>, group?: string) {
  return containerFixture({ id, name: id, state, labels, group });
}

describe("useSwarmStore", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    showAllContainers.value = false;
    containers.value = [
      task(
        "web.1",
        "running",
        {
          "com.docker.stack.namespace": "app",
          "com.docker.swarm.service.name": "app_web",
        },
        "frontend",
      ),
      task(
        "job.1",
        "exited",
        {
          "com.docker.stack.namespace": "batch",
          "com.docker.swarm.service.name": "batch_job",
        },
        "workers",
      ),
      task("gone.1", "deleted", {
        "com.docker.stack.namespace": "old",
        "com.docker.swarm.service.name": "old_svc",
      }),
    ];
  });

  test("lists only running tasks by default", () => {
    const store = useSwarmStore();
    expect(store.stacks.map((s) => s.name)).toEqual(["app"]);
    expect(store.services.map((s) => s.name)).toEqual(["app_web"]);
  });

  test("lists stopped tasks in stacks and services when show stopped is on", () => {
    showAllContainers.value = true;
    const store = useSwarmStore();
    expect(store.stacks.map((s) => s.name)).toEqual(["app", "batch"]);
    expect(store.services.map((s) => s.name)).toEqual(["app_web", "batch_job"]);
  });

  test("keeps custom groups running-only, since their stream cannot open stopped containers", () => {
    showAllContainers.value = true;
    const store = useSwarmStore();
    expect(store.customGroups.map((g) => g.name)).toEqual(["frontend"]);
  });
});
