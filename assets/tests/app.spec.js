import fetchMock from "fetch-mock";
import EventSource from "eventsourcemock";
import flushPromises from "flush-promises";
import { shallowMount } from "@vue/test-utils";
import App from "../App";

describe("App", () => {
  test("is a Vue instance", async () => {
    global.BASE_PATH = "";
    global.EventSource = EventSource;
    fetchMock.get("/api/containers.json", [
      {
        id: "abc",
        name: "Test 1"
      },
      {
        id: "xyz",
        name: "Test 2"
      }
    ]);
    const wrapper = shallowMount(App, {
      stubs: ["router-link", "router-view"]
    });
    expect(wrapper.isVueInstance()).toBeTruthy();
    await flushPromises();
    expect(wrapper.vm.title).toBe("2 containers - Dozzle");
  });
});
