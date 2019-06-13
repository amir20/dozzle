import EventSource from "eventsourcemock";
import { sources } from "eventsourcemock";
import { shallowMount } from "@vue/test-utils";
import Container from "./Container";

describe("<Container />", () => {
  beforeEach(() => {
    global.BASE_PATH = "";
    global.EventSource = EventSource;
  });

  test("is a Vue instance", async () => {
    const wrapper = shallowMount(Container);
    expect(wrapper.isVueInstance()).toBeTruthy();
  });

  test("renders correctly", async () => {
    const wrapper = shallowMount(Container);
    expect(wrapper.element).toMatchSnapshot();
  });

  test("should connect to EventSource", async () => {
    shallowMount(Container, {
      propsData: { id: "abc" }
    });
    sources["/api/logs/stream?id=abc"].emitOpen();
    expect(sources["/api/logs/stream?id=abc"].readyState).toBe(1);
  });

  test("should close EventSource", async () => {
    const wrapper = shallowMount(Container, {
      propsData: { id: "abc" }
    });
    sources["/api/logs/stream?id=abc"].emitOpen();
    wrapper.destroy();
    expect(sources["/api/logs/stream?id=abc"].readyState).toBe(2);
  });

  test("should parse messages", async () => {
    const wrapper = shallowMount(Container, {
      propsData: { id: "abc" }
    });
    sources["/api/logs/stream?id=abc"].emitOpen();
    sources["/api/logs/stream?id=abc"].emitMessage({ data: `2019-06-13T00:55:42.459034602Z "This is a message."` });
    const [{ dateRelative, ...other }, _] = wrapper.vm.messages;

    expect(other).toMatchInlineSnapshot(`
      Object {
        "date": 2019-06-13T00:55:42.459Z,
        "key": 0,
        "message": " \\"This is a message.\\"",
      }
    `);
  });
});
