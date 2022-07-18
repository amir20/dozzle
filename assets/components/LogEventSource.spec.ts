import { mount } from "@vue/test-utils";
import { createTestingPinia } from "@pinia/testing";
// @ts-ignore
import EventSource, { sources } from "eventsourcemock";
import LogEventSource from "./LogEventSource.vue";
import LogViewer from "./LogViewer.vue";
import { settings } from "../composables/settings";
import { useSearchFilter } from "@/composables/search";
import { vi, describe, expect, beforeEach, test, beforeAll, afterAll } from "vitest";
import { computed, Ref } from "vue";
import { createRouter, createWebHistory } from "vue-router";

vi.mock("lodash.debounce", () => ({
  __esModule: true,
  default: vi.fn((fn) => {
    fn.cancel = () => {};
    return fn;
  }),
}));

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "" },
}));

/**
 * @vitest-environment jsdom
 */
describe("<LogEventSource />", () => {
  const search = useSearchFilter();

  beforeEach(() => {
    global.EventSource = EventSource;
    window.scrollTo = vi.fn();
    global.IntersectionObserver = vi.fn().mockImplementation(() => ({
      observe: vi.fn(),
      disconnect: vi.fn(),
    }));
  });

  function createLogEventSource(
    {
      searchFilter = "",
      hourStyle = "auto",
    }: { searchFilter?: string | undefined; hourStyle?: "auto" | "24" | "12" } = {
      hourStyle: "auto",
    }
  ) {
    settings.value.hourStyle = hourStyle;
    search.searchFilter.value = searchFilter;

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
          container: computed(() => ({ id: "abc", image: "test:v123" })),
        },
      },
      slots: {
        default: `
        <template #scoped="params"><log-viewer :messages="params.messages"></log-viewer></template>
        `,
      },
      props: { id: "abc" },
    });
  }

  test("renders correctly", async () => {
    const wrapper = createLogEventSource();
    expect(wrapper.html()).toMatchSnapshot();
  });

  test("should connect to EventSource", async () => {
    const wrapper = createLogEventSource();
    sources["/api/logs/stream?id=abc&lastEventId="].emitOpen();
    expect(sources["/api/logs/stream?id=abc&lastEventId="].readyState).toBe(1);
    wrapper.unmount();
  });

  test("should close EventSource", async () => {
    const wrapper = createLogEventSource();
    sources["/api/logs/stream?id=abc&lastEventId="].emitOpen();
    wrapper.unmount();
    expect(sources["/api/logs/stream?id=abc&lastEventId="].readyState).toBe(2);
  });

  test("should parse messages", async () => {
    const wrapper = createLogEventSource();
    sources["/api/logs/stream?id=abc&lastEventId="].emitOpen();
    sources["/api/logs/stream?id=abc&lastEventId="].emitMessage({
      data: `{"ts":1560336942.459, "m":"This is a message."}`,
    });

    const [message, _] = wrapper.vm.messages;
    expect(message).toMatchSnapshot();
  });

  describe("render html correctly", () => {
    const RealDate = Date;
    beforeAll(() => {
      // @ts-ignore
      global.Date = class extends RealDate {
        constructor(arg: any | number) {
          super(arg);
          if (arg) {
            return new RealDate(arg);
          } else {
            return new RealDate(1560336936000);
          }
        }
      };
    });
    afterAll(() => (global.Date = RealDate));

    test("should render messages", async () => {
      const wrapper = createLogEventSource();
      sources["/api/logs/stream?id=abc&lastEventId="].emitOpen();
      sources["/api/logs/stream?id=abc&lastEventId="].emitMessage({
        data: `{"ts":1560336942.459, "m":"This is a message."}`,
      });

      await wrapper.vm.$nextTick();
      expect(wrapper.find("ul.events").html()).toMatchSnapshot();
    });

    test("should render messages with color", async () => {
      const wrapper = createLogEventSource();
      sources["/api/logs/stream?id=abc&lastEventId="].emitOpen();
      sources["/api/logs/stream?id=abc&lastEventId="].emitMessage({
        data: '{"ts":1560336942.459,"m":"\\u001b[30mblack\\u001b[37mwhite"}',
      });

      await wrapper.vm.$nextTick();
      expect(wrapper.find("ul.events").html()).toMatchSnapshot();
    });

    test("should render messages with html entities", async () => {
      const wrapper = createLogEventSource();
      sources["/api/logs/stream?id=abc&lastEventId="].emitOpen();
      sources["/api/logs/stream?id=abc&lastEventId="].emitMessage({
        data: `{"ts":1560336942.459, "m":"<test>foo bar</test>"}`,
      });

      await wrapper.vm.$nextTick();
      expect(wrapper.find("ul.events").html()).toMatchSnapshot();
    });

    test("should render dates with 12 hour style", async () => {
      const wrapper = createLogEventSource({ hourStyle: "12" });
      sources["/api/logs/stream?id=abc&lastEventId="].emitOpen();
      sources["/api/logs/stream?id=abc&lastEventId="].emitMessage({
        data: `{"ts":1560336942.459, "m":"<test>foo bar</test>"}`,
      });

      await wrapper.vm.$nextTick();
      expect(wrapper.find("ul.events").html()).toMatchSnapshot();
    });

    test("should render dates with 24 hour style", async () => {
      const wrapper = createLogEventSource({ hourStyle: "24" });
      sources["/api/logs/stream?id=abc&lastEventId="].emitOpen();
      sources["/api/logs/stream?id=abc&lastEventId="].emitMessage({
        data: `{"ts":1560336942.459, "m":"<test>foo bar</test>"}`,
      });

      await wrapper.vm.$nextTick();
      expect(wrapper.find("ul.events").html()).toMatchSnapshot();
    });

    test("should render messages with filter", async () => {
      const wrapper = createLogEventSource({ searchFilter: "test" });
      sources["/api/logs/stream?id=abc&lastEventId="].emitOpen();
      sources["/api/logs/stream?id=abc&lastEventId="].emitMessage({
        data: `{"ts":1560336942.459, "m":"<test>foo bar</test>"}`,
      });
      sources["/api/logs/stream?id=abc&lastEventId="].emitMessage({
        data: `{"ts":1560336942.459, "m":"<test>test bar</test>"}`,
      });

      await wrapper.vm.$nextTick();
      expect(wrapper.find("ul.events").html()).toMatchSnapshot();
    });
  });
});
