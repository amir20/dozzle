import EventSource from "eventsourcemock";
import { shallowMount, RouterLinkStub, createLocalVue } from "@vue/test-utils";
import Vuex from "vuex";
import App from "./App";

const localVue = createLocalVue();

localVue.use(Vuex);

describe("<App />", () => {
  const stubs = { RouterLink: RouterLinkStub, "router-view": true };
  let store;

  beforeEach(() => {
    global.BASE_PATH = "";
    global.EventSource = EventSource;
    const state = {
      containers: [
        { id: "abc", name: "Test 1" },
        { id: "xyz", name: "Test 2" }
      ]
    };

    const actions = {
      FETCH_CONTAINERS: () => Promise.resolve()
    };

    store = new Vuex.Store({
      state,
      actions
    });
  });

  test("is a Vue instance", async () => {
    const wrapper = shallowMount(App, { stubs, store, localVue });
    expect(wrapper.isVueInstance()).toBeTruthy();
  });

  test("has right title", async () => {
    const wrapper = shallowMount(App, { stubs, store, localVue });
    await wrapper.vm.$nextTick();
    expect(wrapper.vm.title).toContain("2 containers");
  });

  test("renders correctly", async () => {
    const wrapper = shallowMount(App, { stubs, store, localVue });
    await wrapper.vm.$nextTick();
    expect(wrapper.element).toMatchSnapshot();
  });
});
