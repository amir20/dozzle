import EventSource from "eventsourcemock";
import { sources } from "eventsourcemock";
import { shallowMount, mount, createLocalVue } from "@vue/test-utils";
import Vuex from "vuex";
import MockDate from "mockdate";
import debounce from "lodash.debounce";
import LogEventSource from "./LogEventSource.vue";
import LogViewer from "./LogViewer.vue";

jest.mock("lodash.debounce", () => jest.fn(fn => fn));

describe("<LogEventSource />", () => {
  beforeEach(() => {
    global.BASE_PATH = "";
    global.EventSource = EventSource;
    MockDate.set("6/12/2019", 0);
    window.scrollTo = jest.fn();

    const observe = jest.fn();
    const unobserve = jest.fn();
    global.IntersectionObserver = jest.fn(() => ({
      observe,
      unobserve
    }));
    debounce.mockClear();
  });

  afterEach(() => MockDate.reset());

  function createLogEventSource(searchFilter = null) {
    const localVue = createLocalVue();
    localVue.use(Vuex);

    localVue.component("log-event-source", LogEventSource);
    localVue.component("log-viewer", LogViewer);

    const state = { searchFilter, settings: { size: "medium" } };

    const store = new Vuex.Store({
      state
    });

    return mount(LogEventSource, {
      localVue,
      store,
      scopedSlots: {
        default: `
        <log-viewer :messages="props.messages"></log-viewer>
        `
      },
      propsData: { id: "abc" }
    });
  }

  test("is a Vue instance", async () => {
    const wrapper = shallowMount(LogEventSource);
    expect(wrapper.isVueInstance()).toBeTruthy();
  });

  test("renders correctly", async () => {
    const wrapper = createLogEventSource();
    expect(wrapper.element).toMatchInlineSnapshot(`
      <div>
        <div
          class="control"
        />
         
        <ul
          class="events medium"
        />
      </div>
    `);
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
        "message": "\\"This is a message.\\"",
      }
    `);
  });

  test("should pass messages to slot", async () => {
    const wrapper = createLogEventSource();
    sources["/api/logs/stream?id=abc"].emitOpen();
    sources["/api/logs/stream?id=abc"].emitMessage({ data: `2019-06-12T10:55:42.459034602Z "This is a message."` });
    const [message, _] = wrapper.find(LogViewer).vm.messages;

    const { key, ...messageWithoutKey } = message;

    expect(key).toBe("2019-06-12T10:55:42.459034602Z");

    expect(messageWithoutKey).toMatchInlineSnapshot(`
      Object {
        "date": 2019-06-12T10:55:42.459Z,
        "message": "\\"This is a message.\\"",
      }
    `);
  });

  test("should render messages", async () => {
    const wrapper = createLogEventSource();
    sources["/api/logs/stream?id=abc"].emitOpen();
    sources["/api/logs/stream?id=abc"].emitMessage({ data: `2019-06-12T10:55:42.459034602Z "This is a message."` });

    expect(wrapper.find("ul.events")).toMatchInlineSnapshot(`
      <ul class="events medium">
        <li><span class="date">today at 10:55 AM</span> <span class="text">"This is a message."</span></li>
      </ul>
    `);
  });

  test("should render messages with color", async () => {
    const wrapper = createLogEventSource();
    sources["/api/logs/stream?id=abc"].emitOpen();
    sources["/api/logs/stream?id=abc"].emitMessage({
      data: `2019-06-12T10:55:42.459034602Z \x1b[30mblack\x1b[37mwhite`
    });

    expect(wrapper.find("ul.events")).toMatchInlineSnapshot(`
      <ul class="events medium">
        <li><span class="date">today at 10:55 AM</span> <span class="text"><span style="color:#000">black<span style="color:#AAA">white</span></span></span></li>
      </ul>
    `);
  });

  test("should render messages with html entities", async () => {
    const wrapper = createLogEventSource();
    sources["/api/logs/stream?id=abc"].emitOpen();
    sources["/api/logs/stream?id=abc"].emitMessage({
      data: `2019-06-12T10:55:42.459034602Z <test>foo bar</test>`
    });

    expect(wrapper.find("ul.events")).toMatchInlineSnapshot(`
      <ul class="events medium">
        <li><span class="date">today at 10:55 AM</span> <span class="text">&lt;test&gt;foo bar&lt;/test&gt;</span></li>
      </ul>
    `);
  });

  test("should render messages with filter", async () => {
    const wrapper = createLogEventSource("test");
    sources["/api/logs/stream?id=abc"].emitOpen();
    sources["/api/logs/stream?id=abc"].emitMessage({
      data: `2019-06-11T10:55:42.459034602Z Foo bar`
    });
    sources["/api/logs/stream?id=abc"].emitMessage({
      data: `2019-06-12T10:55:42.459034602Z This is a test <hi></hi>`
    });

    expect(wrapper.find("ul.events")).toMatchInlineSnapshot(`
      <ul class="events medium">
        <li><span class="date">today at 10:55 AM</span> <span class="text">This is a <mark>test</mark> &lt;hi&gt;&lt;/hi&gt;</span></li>
      </ul>
    `);
  });
});
