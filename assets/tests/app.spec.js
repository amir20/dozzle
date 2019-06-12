import fetchMock from "fetch-mock";
import EventSource from "eventsourcemock";
import { shallowMount } from "@vue/test-utils";
import App from "../App";

describe("<App />", () => {
  const stubs = ["router-link", "router-view"];
  beforeEach(() => {
    global.BASE_PATH = "";
    global.EventSource = EventSource;
    fetchMock.getOnce("/api/containers.json", [{ id: "abc", name: "Test 1" }, { id: "xyz", name: "Test 2" }]);
  });
  afterEach(() => fetchMock.reset());

  test("is a Vue instance", async () => {
    const wrapper = shallowMount(App, { stubs });
    expect(wrapper.isVueInstance()).toBeTruthy();
  });

  test("has right title", async () => {
    const wrapper = shallowMount(App, { stubs });
    await fetchMock.flush();
    expect(wrapper.vm.title).toBe("2 containers - Dozzle");
  });

  test("renders correctly", async () => {
    const wrapper = shallowMount(App, { stubs });
    await fetchMock.flush();
    expect(wrapper.element).toMatchSnapshot();
  });
});
