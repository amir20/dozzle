import { createTestingPinia } from "@pinia/testing";
import { mount } from "@vue/test-utils";
import { useSearchFilter } from "@/composable/search";
import { settings } from "@/stores/settings";
// @ts-ignore
import EventSource, { sources } from "eventsourcemock";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { computed, nextTick } from "vue";
import { createI18n } from "vue-i18n";
import { createRouter, createWebHistory } from "vue-router";
import { default as Component } from "./EventSource.vue";
import LogViewer from "@/components/LogViewer/LogViewer.vue";
import { Container } from "@/models/Container";
import { Level } from "@/models/LogEntry";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [{ name: "localhost", id: "localhost" }] },
  withBase: (path: string) => path,
}));

/**
 * @vitest-environment jsdom
 */
describe("<ContainerEventSource />", () => {
  const search = useSearchFilter();

  beforeEach(() => {
    global.EventSource = EventSource;
    // @ts-ignore
    window.scrollTo = vi.fn();
    global.IntersectionObserver = vi.fn().mockImplementation(() => ({
      observe: vi.fn(),
      disconnect: vi.fn(),
    }));
    vi.useFakeTimers();
    vi.setSystemTime(1560336942459);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  function createLogEventSource(
    {
      searchFilter = "",
      hourStyle = "auto",
    }: { searchFilter?: string | undefined; hourStyle?: "auto" | "24" | "12" } = {
      hourStyle: "auto",
    },
  ) {
    settings.value.hourStyle = hourStyle;
    search.searchQueryFilter.value = searchFilter;
    if (searchFilter) {
      search.showSearch.value = true;
    }

    const router = createRouter({
      history: createWebHistory("/"),
      routes: [
        {
          path: "/",
          component: {
            template: "Test from createLogEventSource",
          },
        },
        {
          name: "/container/[id].time.[datetime]",
          path: "/container/:id/time/:datetime",
          component: {
            template: "Test from createLogEventSource",
          },
        },
      ],
    });

    return mount(Component, {
      global: {
        plugins: [
          router,
          createTestingPinia({
            createSpy: vi.fn,
            stubActions: false,
            initialState: {
              container: { containers: [{ id: "abc", image: "test:v123", host: "localhost" }] },
            },
          }),
          createI18n({}),
        ],
        components: {
          LogViewer,
        },
        provide: {
          [scrollContextKey as symbol]: {
            paused: computed(() => false),
            loading: computed(() => false),
          },
          [loggingContextKey as symbol]: {
            containers: computed(() => [{ id: "abc", image: "test:v123", host: "localhost" }]),
            streamConfig: reactive({ stdout: true, stderr: true }),
            hasComplexLogs: ref(false),
            levels: new Set<Level>(["info"]),
            historical: ref(false),
          },
        },
      },
      slots: {
        default: `
        <template #scoped="params"><LogViewer :messages="params.messages" :show-container-name="false" :visible-keys="[]" /></template>
        `,
      },
      props: {
        streamSource: useContainerStream,
        entity: new Container(
          "abc",
          new Date(), // created
          new Date(), // started
          new Date(), // finished
          "image",
          "name",
          "command",
          "localhost",
          {},
          "created",
          0,
          0,
          [],
        ),
      },
    });
  }

  const sourceUrl = "/api/hosts/localhost/containers/abc/logs/stream?stdout=1&stderr=1&levels=info";

  test("renders loading correctly", async () => {
    const wrapper = createLogEventSource();
    expect(wrapper.find("ul.animate-pulse").exists()).toBe(true);
  });

  test("should connect to EventSource", async () => {
    const wrapper = createLogEventSource();
    sources[sourceUrl].emitOpen();
    expect(sources[sourceUrl].readyState).toBe(1);
    wrapper.unmount();
  });

  test("should close EventSource", async () => {
    const wrapper = createLogEventSource();
    sources[sourceUrl].emitOpen();
    wrapper.unmount();
    expect(sources[sourceUrl].readyState).toBe(2);
  });

  test("should parse messages", async () => {
    const wrapper = createLogEventSource();
    sources[sourceUrl].emitOpen();
    sources[sourceUrl].emitMessage({
      data: `{"ts":1560336942459, "m":"This is a message.", "id":1, "rm": "This is a message.", "c": "abc"}`,
    });

    vi.runAllTimers();
    await nextTick();

    // @ts-ignore
    const [message, _] = wrapper.vm.messages;
    expect(message).toMatchSnapshot();
  });

  describe("render html correctly", () => {
    test("should render messages", async () => {
      const wrapper = createLogEventSource();
      sources[sourceUrl].emitOpen();
      sources[sourceUrl].emitMessage({
        data: `{"ts":1560336942459, "m":"This is a message.", "id":1, "rm": "This is a message.", "c": "abc"}`,
      });

      vi.runAllTimers();
      await nextTick();

      expect(wrapper.find("ul[data-logs]").html()).toMatchSnapshot();
    });

    test("should render dates with 12 hour style", async () => {
      const wrapper = createLogEventSource({ hourStyle: "12" });
      sources[sourceUrl].emitOpen();
      sources[sourceUrl].emitMessage({
        data: `{"ts":1560336942459, "m":"foo bar", "id":1, "rm": "foo bar", "c": "abc"}`,
      });

      vi.runAllTimers();
      await nextTick();

      expect(wrapper.find("ul[data-logs]").html()).toMatchSnapshot();
    });

    test("should render dates with 24 hour style", async () => {
      const wrapper = createLogEventSource({ hourStyle: "24" });
      sources[sourceUrl].emitOpen();
      sources[sourceUrl].emitMessage({
        data: `{"ts":1560336942459, "m":"foo bar", "id":1, "c": "abc"}`,
      });

      vi.runAllTimers();
      await nextTick();

      expect(wrapper.find("ul[data-logs]").html()).toMatchSnapshot();
    });
  });
});
