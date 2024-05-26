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
    search.searchFilter.value = searchFilter;
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
      ],
    });

    return mount(Component<Container>, {
      global: {
        plugins: [router, createTestingPinia({ createSpy: vi.fn }), createI18n({})],
        components: {
          LogViewer,
        },
        provide: {
          scrollingPaused: computed(() => false),
          [loggingContextKey as symbol]: {
            containers: computed(() => [{ id: "abc", image: "test:v123", host: "localhost" }]),
            streamConfig: reactive({ stdout: true, stderr: true }),
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
        entity: new Container("abc", new Date(), "image", "name", "command", "localhost", {}, "status", "created", []),
      },
    });
  }

  const sourceUrl = "/api/hosts/localhost/containers/abc/logs/stream?stdout=1&stderr=1";

  test("renders loading correctly", async () => {
    const wrapper = createLogEventSource();
    expect(wrapper.html()).toMatchSnapshot();
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
      data: `{"ts":1560336942459, "m":"This is a message.", "id":1}`,
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
        data: `{"ts":1560336942459, "m":"This is a message.", "id":1}`,
      });

      vi.runAllTimers();
      await nextTick();

      expect(wrapper.find("ul.events").html()).toMatchSnapshot();
    });

    test("should render messages with html entities", async () => {
      const wrapper = createLogEventSource();
      sources[sourceUrl].emitOpen();
      sources[sourceUrl].emitMessage({
        data: `{"ts":1560336942459, "m":"<test>foo bar</test>", "id":1}`,
      });

      vi.runAllTimers();
      await nextTick();

      expect(wrapper.find("ul.events").html()).toMatchSnapshot();
    });

    test("should render dates with 12 hour style", async () => {
      const wrapper = createLogEventSource({ hourStyle: "12" });
      sources[sourceUrl].emitOpen();
      sources[sourceUrl].emitMessage({
        data: `{"ts":1560336942459, "m":"<test>foo bar</test>", "id":1}`,
      });

      vi.runAllTimers();
      await nextTick();

      expect(wrapper.find("ul.events").html()).toMatchSnapshot();
    });

    test("should render dates with 24 hour style", async () => {
      const wrapper = createLogEventSource({ hourStyle: "24" });
      sources[sourceUrl].emitOpen();
      sources[sourceUrl].emitMessage({
        data: `{"ts":1560336942459, "m":"<test>foo bar</test>", "id":1}`,
      });

      vi.runAllTimers();
      await nextTick();

      expect(wrapper.find("ul.events").html()).toMatchSnapshot();
    });

    test("should render messages with filter", async () => {
      const wrapper = createLogEventSource({ searchFilter: "test" });
      sources[sourceUrl].emitOpen();
      sources[sourceUrl].emitMessage({
        data: `{"ts":1560336942459, "m":"foo bar", "id":1}`,
      });
      sources[sourceUrl].emitMessage({
        data: `{"ts":1560336942459, "m":"test bar", "id":2}`,
      });

      vi.runAllTimers();
      await nextTick();

      expect(wrapper.find("ul.events").html()).toMatchSnapshot();
    });
  });
});
