import debounce from "lodash.debounce";
import EventSource from "eventsourcemock";
import { sources } from "eventsourcemock";
import { shallowMount, mount, createLocalVue } from "@vue/test-utils";
import Vuex from "vuex";
import LogEventSource from "./LogEventSource.vue";
import LogViewer from "./LogViewer.vue";

jest.mock("lodash.debounce", () =>
  jest.fn((fn) => {
    return fn;
  })
);

jest.mock("../store/config.js", () => ({ base: "" }));

describe("<LogEventSource />", () => {
  beforeEach(() => {
    global.EventSource = EventSource;
    window.scrollTo = jest.fn();
    const observe = jest.fn();
    const disconnect = jest.fn();
    global.IntersectionObserver = jest.fn(() => ({
      observe,
      disconnect,
    }));
    debounce.mockClear();
  });

  function createLogEventSource(searchFilter = null) {
    const localVue = createLocalVue();
    localVue.use(Vuex);

    localVue.component("log-viewer", LogViewer);

    const state = { searchFilter, settings: { size: "medium", showTimestamp: true } };

    const store = new Vuex.Store({
      state,
    });

    return mount(LogEventSource, {
      localVue,
      store,
      scopedSlots: {
        default: `
        <log-viewer :messages="props.messages"></log-viewer>
        `,
      },
      propsData: { id: "abc" },
    });
  }

  test("renders correctly", async () => {
    const wrapper = createLogEventSource();
    expect(wrapper.element).toMatchSnapshot();
  });

  test("should connect to EventSource", async () => {
    shallowMount(LogEventSource);
    sources["/api/logs/stream?id=abc"].emitOpen();
    expect(sources["/api/logs/stream?id=abc"].readyState).toBe(1);
  });

  test("should close EventSource", async () => {
    const wrapper = createLogEventSource();
    sources["/api/logs/stream?id=abc"].emitOpen();
    wrapper.destroy();
    expect(sources["/api/logs/stream?id=abc"].readyState).toBe(2);
  });

  test("should parse messages", async () => {
    const wrapper = createLogEventSource();
    sources["/api/logs/stream?id=abc"].emitOpen();
    sources["/api/logs/stream?id=abc"].emitMessage({ data: `2019-06-12T10:55:42.459034602Z "This is a message."` });

    const [message, _] = wrapper.vm.messages;
    const { key, ...messageWithoutKey } = message;

    expect(key).toBe("2019-06-12T10:55:42.459034602Z");
    expect(messageWithoutKey).toMatchInlineSnapshot(`
      Object {
        "date": 2019-06-12T10:55:42.459Z,
        "message": " \\"This is a message.\\"",
      }
    `);
  });

  test("should parse messages with loki's timestamp format", async () => {
    const wrapper = createLogEventSource();
    sources["/api/logs/stream?id=abc"].emitOpen();
    sources["/api/logs/stream?id=abc"].emitMessage({ data: `2020-04-27T12:35:43.272974324+02:00 xxxxx` });

    const [message, _] = wrapper.vm.messages;
    const { key, ...messageWithoutKey } = message;

    expect(key).toBe("2020-04-27T12:35:43.272974324+02:00");
    expect(messageWithoutKey).toMatchInlineSnapshot(`
      Object {
        "date": 2020-04-27T10:35:43.272Z,
        "message": " xxxxx",
      }
    `);
  });

  test("should pass messages to slot", async () => {
    const wrapper = createLogEventSource();
    sources["/api/logs/stream?id=abc"].emitOpen();
    sources["/api/logs/stream?id=abc"].emitMessage({ data: `2019-06-12T10:55:42.459034602Z "This is a message."` });
    const [message, _] = wrapper.findComponent(LogViewer).vm.messages;

    const { key, ...messageWithoutKey } = message;

    expect(key).toBe("2019-06-12T10:55:42.459034602Z");

    expect(messageWithoutKey).toMatchInlineSnapshot(`
      Object {
        "date": 2019-06-12T10:55:42.459Z,
        "message": " \\"This is a message.\\"",
      }
    `);
  });

  describe("render html correctly", () => {
    const RealDate = Date;
    beforeAll(() => {
      global.Date = class extends (
        RealDate
      ) {
        constructor(arg) {
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
      sources["/api/logs/stream?id=abc"].emitOpen();
      sources["/api/logs/stream?id=abc"].emitMessage({ data: `2019-06-12T10:55:42.459034602Z "This is a message."` });

      await wrapper.vm.$nextTick();
      expect(wrapper.find("ul.events")).toMatchInlineSnapshot(`
        <ul class="events medium">
          <li class=""><span class="date"><time datetime="2019-06-12T10:55:42.459Z">today at 10:55 AM</time></span> <span class="text"> "This is a message."</span></li>
        </ul>
      `);
    });

    test("should render messages with color", async () => {
      const wrapper = createLogEventSource();
      sources["/api/logs/stream?id=abc"].emitOpen();
      sources["/api/logs/stream?id=abc"].emitMessage({
        data: `2019-06-12T10:55:42.459034602Z \x1b[30mblack\x1b[37mwhite`,
      });

      await wrapper.vm.$nextTick();
      expect(wrapper.find("ul.events")).toMatchInlineSnapshot(`
        <ul class="events medium">
          <li class=""><span class="date"><time datetime="2019-06-12T10:55:42.459Z">today at 10:55 AM</time></span> <span class="text"> <span style="color:#000">black<span style="color:#AAA">white</span></span></span></li>
        </ul>
      `);
    });

    test("should render messages with html entities", async () => {
      const wrapper = createLogEventSource();
      sources["/api/logs/stream?id=abc"].emitOpen();
      sources["/api/logs/stream?id=abc"].emitMessage({
        data: `2019-06-12T10:55:42.459034602Z <test>foo bar</test>`,
      });

      await wrapper.vm.$nextTick();
      expect(wrapper.find("ul.events")).toMatchInlineSnapshot(`
        <ul class="events medium">
          <li class=""><span class="date"><time datetime="2019-06-12T10:55:42.459Z">today at 10:55 AM</time></span> <span class="text"> &lt;test&gt;foo bar&lt;/test&gt;</span></li>
        </ul>
      `);
    });

    test("should render messages with filter", async () => {
      const wrapper = createLogEventSource("test");
      sources["/api/logs/stream?id=abc"].emitOpen();
      sources["/api/logs/stream?id=abc"].emitMessage({
        data: `2019-06-11T10:55:42.459034602Z Foo bar`,
      });
      sources["/api/logs/stream?id=abc"].emitMessage({
        data: `2019-06-12T10:55:42.459034602Z This is a test <hi></hi>`,
      });

      await wrapper.vm.$nextTick();
      expect(wrapper.find("ul.events")).toMatchInlineSnapshot(`
        <ul class="events medium">
          <li class=""><span class="date"><time datetime="2019-06-12T10:55:42.459Z">today at 10:55 AM</time></span> <span class="text"> This is a <mark>test</mark> &lt;hi&gt;&lt;/hi&gt;</span></li>
        </ul>
      `);
    });
  });
});
