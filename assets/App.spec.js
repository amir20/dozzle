import fetchMock from "fetch-mock";
import EventSource from "eventsourcemock";
import { shallowMount, RouterLinkStub } from "@vue/test-utils";
import App from "./App";

describe("<App />", () => {
  const stubs = { RouterLink: RouterLinkStub, "router-view": true };
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
    expect(wrapper.vm.title).toContain("2 containers");
  });

  test("renders correctly", async () => {
    const wrapper = shallowMount(App, { stubs });
    await fetchMock.flush();
    expect(wrapper.element).toMatchSnapshot();
  });

  test("renders router-link correctly", async () => {
    const wrapper = shallowMount(App, { stubs });
    await fetchMock.flush();
    expect(wrapper.find(RouterLinkStub).props().to).toMatchInlineSnapshot(`
      Object {
        "name": "container",
        "params": Object {
          "id": "abc",
          "name": "Test 1",
        },
      }
    `);
  });
});
