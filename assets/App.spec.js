import EventSource from "eventsourcemock";
import { shallowMount, RouterLinkStub, createLocalVue } from "@vue/test-utils";
import Vuex from "vuex";
import App from "./App";

jest.mock("./store/config.js", () => ({ base: "" }));

const localVue = createLocalVue();

localVue.use(Vuex);

describe("<App />", () => {
  const stubs = { RouterLink: RouterLinkStub, "router-view": true, icon: true };
  let store;

  beforeEach(() => {
    global.EventSource = EventSource;
    const state = {
      settings: { menuWidth: 15 },
    };

    const getters = {
      visibleContainers() {
        return [
          { id: "abc", name: "Test 1" },
          { id: "xyz", name: "Test 2" },
        ];
      },
      activeContainers() {
        return [];
      },
    };

    store = new Vuex.Store({
      state,
      getters,
    });
  });

  test("has right title", async () => {
    const wrapper = shallowMount(App, { stubs, store, localVue });
    await wrapper.vm.$nextTick();
    wrapper.vm.$options.watch.visibleContainers.call(wrapper.vm);

    expect(wrapper.vm.title).toContain("2 containers");
  });

  test("renders correctly", async () => {
    const wrapper = shallowMount(App, { stubs, store, localVue });
    await wrapper.vm.$nextTick();
    expect(wrapper.element).toMatchSnapshot();
  });
});
