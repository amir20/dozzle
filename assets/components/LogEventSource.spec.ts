import { mount } from "@vue/test-utils";
import { createStore } from "vuex";
// @ts-ignore
import EventSource, { sources } from "eventsourcemock";
import debounce from "lodash.debounce";
import LogEventSource from "./LogEventSource.vue";
import LogViewer from "./LogViewer.vue";
import { settings } from "../composables/settings";
import { mocked } from "ts-jest/utils";

jest.mock("lodash.debounce", () =>
  jest.fn((fn) => {
    fn.cancel = () => {};
    return fn;
  })
);

jest.mock("../store/config.ts", () => ({ base: "" }));

describe("<LogEventSource />", () => {
  beforeEach(() => {
    // @ts-ignore
    global.EventSource = EventSource;
    window.scrollTo = jest.fn();
    global.IntersectionObserver = jest.fn().mockImplementation(() => ({
      observe: jest.fn(),
      disconnect: jest.fn(),
    }));

    mocked(debounce).mockClear();
    jest.resetModules();
  });

  function createLogEventSource(
    { searchFilter = null, hourStyle = "auto" }: { searchFilter?: string | null; hourStyle?: "auto" | "24" | "12" } = {
      hourStyle: "auto",
    }
  ) {
    settings.value.hourStyle = hourStyle;
    const store = createStore({
      state: { searchFilter },
      getters: {
        allContainersById() {
          return {
            abc: { state: "running" },
          };
        },
      },
    });

    return mount(LogEventSource, {
      global: {
        plugins: [store],
        components: {
          LogViewer,
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
      data: `2019-06-12T10:55:42.459034602Z "This is a message."`,
    });

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

  test("should parse messages with loki's timestamp format", async () => {
    const wrapper = createLogEventSource();
    sources["/api/logs/stream?id=abc&lastEventId="].emitOpen();
    sources["/api/logs/stream?id=abc&lastEventId="].emitMessage({ data: `2020-04-27T12:35:43.272974324+02:00 xxxxx` });

    const [message, _] = wrapper.vm.messages;
    const { key, ...messageWithoutKey } = message;

    expect(key).toBe("2020-04-27T12:35:43.272974324+02:00");
    expect(messageWithoutKey).toMatchInlineSnapshot(`
      Object {
        "date": 2020-04-27T10:35:43.272Z,
        "message": "xxxxx",
      }
    `);
  });

  test("should pass messages to slot", async () => {
    const wrapper = createLogEventSource();
    sources["/api/logs/stream?id=abc&lastEventId="].emitOpen();
    sources["/api/logs/stream?id=abc&lastEventId="].emitMessage({
      data: `2019-06-12T10:55:42.459034602Z "This is a message."`,
    });
    const [message, _] = wrapper.getComponent(LogViewer).vm.messages;

    const { key, ...messageWithoutKey } = message;

    expect(key).toBe("2019-06-12T10:55:42.459034602Z");

    expect(messageWithoutKey).toMatchInlineSnapshot(`
      Object {
        "date": 2019-06-12T10:55:42.459Z,
        "message": "\\"This is a message.\\"",
      }
    `);
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
        data: `2019-06-12T10:55:42.459034602Z "This is a message."`,
      });

      await wrapper.vm.$nextTick();
      expect(wrapper.find("ul.events").html()).toMatchInlineSnapshot(
        `"<ul class=\\"events medium\\"><li><span class=\\"date\\"><time datetime=\\"2019-06-12T10:55:42.459Z\\">today at 10:55:42 AM</time></span><span class=\\"text\\">\\"This is a message.\\"</span></li></ul>"`
      );
    });

    test("should render messages with color", async () => {
      const wrapper = createLogEventSource();
      sources["/api/logs/stream?id=abc&lastEventId="].emitOpen();
      sources["/api/logs/stream?id=abc&lastEventId="].emitMessage({
        data: `2019-06-12T10:55:42.459034602Z \x1b[30mblack\x1b[37mwhite`,
      });

      await wrapper.vm.$nextTick();
      expect(wrapper.find("ul.events").html()).toMatchInlineSnapshot(
        `"<ul class=\\"events medium\\"><li><span class=\\"date\\"><time datetime=\\"2019-06-12T10:55:42.459Z\\">today at 10:55:42 AM</time></span><span class=\\"text\\"><span style=\\"color:#000\\">black<span style=\\"color:#AAA\\">white</span></span></span></li></ul>"`
      );
    });

    test("should render messages with html entities", async () => {
      const wrapper = createLogEventSource();
      sources["/api/logs/stream?id=abc&lastEventId="].emitOpen();
      sources["/api/logs/stream?id=abc&lastEventId="].emitMessage({
        data: `2019-06-12T10:55:42.459034602Z <test>foo bar</test>`,
      });

      await wrapper.vm.$nextTick();
      expect(wrapper.find("ul.events").html()).toMatchInlineSnapshot(
        `"<ul class=\\"events medium\\"><li><span class=\\"date\\"><time datetime=\\"2019-06-12T10:55:42.459Z\\">today at 10:55:42 AM</time></span><span class=\\"text\\">&lt;test&gt;foo bar&lt;/test&gt;</span></li></ul>"`
      );
    });

    test("should render dates with 12 hour style", async () => {
      const wrapper = createLogEventSource({ hourStyle: "12" });
      sources["/api/logs/stream?id=abc&lastEventId="].emitOpen();
      sources["/api/logs/stream?id=abc&lastEventId="].emitMessage({
        data: `2019-06-12T23:55:42.459034602Z <test>foo bar</test>`,
      });

      await wrapper.vm.$nextTick();
      expect(wrapper.find("ul.events").html()).toMatchInlineSnapshot(
        `"<ul class=\\"events medium\\"><li><span class=\\"date\\"><time datetime=\\"2019-06-12T23:55:42.459Z\\">today at 11:55:42 PM</time></span><span class=\\"text\\">&lt;test&gt;foo bar&lt;/test&gt;</span></li></ul>"`
      );
    });

    test("should render dates with 24 hour style", async () => {
      const wrapper = createLogEventSource({ hourStyle: "24" });
      sources["/api/logs/stream?id=abc&lastEventId="].emitOpen();
      sources["/api/logs/stream?id=abc&lastEventId="].emitMessage({
        data: `2019-06-12T23:55:42.459034602Z <test>foo bar</test>`,
      });

      await wrapper.vm.$nextTick();
      expect(wrapper.find("ul.events").html()).toMatchInlineSnapshot(
        `"<ul class=\\"events medium\\"><li><span class=\\"date\\"><time datetime=\\"2019-06-12T23:55:42.459Z\\">today at 23:55:42</time></span><span class=\\"text\\">&lt;test&gt;foo bar&lt;/test&gt;</span></li></ul>"`
      );
    });

    test("should render messages with filter", async () => {
      const wrapper = createLogEventSource({ searchFilter: "test" });
      sources["/api/logs/stream?id=abc&lastEventId="].emitOpen();
      sources["/api/logs/stream?id=abc&lastEventId="].emitMessage({
        data: `2019-06-11T10:55:42.459034602Z Foo bar`,
      });
      sources["/api/logs/stream?id=abc&lastEventId="].emitMessage({
        data: `2019-06-12T10:55:42.459034602Z This is a test <hi></hi>`,
      });

      await wrapper.vm.$nextTick();
      expect(wrapper.find("ul.events").html()).toMatchInlineSnapshot(
        `"<ul class=\\"events medium\\"><li><span class=\\"date\\"><time datetime=\\"2019-06-12T10:55:42.459Z\\">today at 10:55:42 AM</time></span><span class=\\"text\\">This is a <mark>test</mark> &lt;hi&gt;&lt;/hi&gt;</span></li></ul>"`
      );
    });
  });
});
