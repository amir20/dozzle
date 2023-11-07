import { mount } from "@vue/test-utils";
import { createTestingPinia } from "@pinia/testing";
// @ts-ignore
import EventSource, { sources } from "eventsourcemock";
import LogEventSource from "./LogEventSource.vue";
import LogViewer from "./LogViewer.vue";
import { settings } from "@/stores/settings";
import { useSearchFilter } from "@/composable/search";
import { vi, describe, expect, beforeEach, test, afterEach } from "vitest";
import { computed, nextTick } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import { containerContext } from "@/composable/containerContext";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [{ name: "localhost", id: "localhost" }] },
  withBase: (path: string) => path,
}));

/**
 * @vitest-environment jsdom
 */
describe("<LogEventSource />", () => {
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

    return mount(LogEventSource, {
      global: {
        plugins: [router, createTestingPinia({ createSpy: vi.fn })],
        components: {
          LogViewer,
        },
        provide: {
          [containerContext as symbol]: {
            container: computed(() => ({ id: "abc", image: "test:v123", host: "localhost" })),
            streamConfig: reactive({ stdout: true, stderr: true }),
          },
          scrollingPaused: computed(() => false),
        },
      },
      slots: {
        default: `
        <template #scoped="params"><log-viewer :messages="params.messages"></log-viewer></template>
        `,
      },
      props: {},
    });
  }

  const sourceUrl = "/api/logs/stream/localhost/abc?stdout=1&stderr=1";

  test("renders correctly", async () => {
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

    test("should render messages with color", async () => {
      const wrapper = createLogEventSource();
      sources[sourceUrl].emitOpen();
      sources[sourceUrl].emitMessage({
        data: '{"ts":1560336942459,"m":"\\u001b[30mblack\\u001b[37mwhite", "id":1}',
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
