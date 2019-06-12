import fetchMock from "fetch-mock";
import EventSource from "eventsourcemock";
import { shallowMount } from "@vue/test-utils";
import App from "../App";

describe("<App />", () => {
  beforeEach(() => {
    global.BASE_PATH = "";
    global.EventSource = EventSource;
  });
  afterEach(() => fetchMock.reset());
  test("is a Vue instance", async () => {
    fetchMock.getOnce("/api/containers.json", [{ id: "abc", name: "Test 1" }, { id: "xyz", name: "Test 2" }]);
    const wrapper = shallowMount(App, {
      stubs: ["router-link", "router-view"]
    });
    expect(wrapper.isVueInstance()).toBeTruthy();
  });

  test("has right title", async () => {
    fetchMock.getOnce("/api/containers.json", [{ id: "abc", name: "Test 1" }, { id: "xyz", name: "Test 2" }]);
    const wrapper = shallowMount(App, {
      stubs: ["router-link", "router-view"]
    });
    await fetchMock.flush();
    expect(wrapper.vm.title).toBe("2 containers - Dozzle");
  });
});
