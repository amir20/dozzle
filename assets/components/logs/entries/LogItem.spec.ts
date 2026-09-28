/** @vitest-environment jsdom */
import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { describe, expect, test, vi } from "vitest";
import { reactive, type Ref } from "vue";
import { Container } from "@/models/Container";
import { SimpleLogEntry } from "@/models/LogEntry";
import { allLevels, loggingContextKey } from "@/composable/logs/logContext";
import { useContainerStore } from "@/stores/container";
import LogItem from "./LogItem.vue";

vi.mock("@/stores/config", () => ({
  default: { base: "", hosts: [{ id: "host", name: "test host" }] },
  withBase: (path: string) => path,
}));

vi.mock("@/stores/container", async () => {
  const { defineStore } = await import("pinia");
  const { shallowRef, computed } = await import("vue");
  return {
    useContainerStore: defineStore("log-item-test-containers", () => {
      const containers = shallowRef<Container[]>([]);
      const currentContainer = (id: Ref<string>) => computed(() => containers.value.find((c) => c.id === id.value));
      return { containers, currentContainer };
    }),
  };
});

function mountItem(containerID: string) {
  setActivePinia(createPinia());
  const store = useContainerStore();
  store.containers = [
    new Container("known", new Date(), new Date(), new Date(), "nginx", "web", "", "host", {}, "running", 0, 0, []),
  ];
  const entry = new SimpleLogEntry("hello", containerID, 1, new Date(), "info", "stdout", "hello");
  return mount(LogItem, {
    props: { logEntry: entry },
    global: {
      stubs: { LogActions: true, LogDate: true, LogStd: true },
      provide: {
        [loggingContextKey as symbol]: reactive({
          streamConfig: { stdout: true, stderr: true },
          containers: [],
          loadingMore: false,
          hasComplexLogs: false,
          levels: new Set(allLevels),
          showContainerName: true,
          showHostname: true,
          historical: false,
        }),
      },
    },
  });
}

describe("<LogItem />", () => {
  test("names the container and host a line came from", () => {
    const text = mountItem("known").text();
    expect(text).toContain("web");
    expect(text).toContain("test host");
  });

  // An update replaces a container under a new id while the lines it already
  // wrote stay in a host or merged view.
  test("renders a line whose container is gone, labelled by its id", () => {
    expect(mountItem("replaced123").text()).toContain("replaced123");
  });
});
