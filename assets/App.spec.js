import EventSource from "eventsourcemock";
import { shallowMount, RouterLinkStub, createLocalVue } from "@vue/test-utils";
import Vuex from "vuex";
import App from "./App";

jest.mock("./store/config.js", () => ({ base: "" }));

jest.mock("~icons/octicon/download-24", () => {}, { virtual: true });
jest.mock("~icons/octicon/trash-24", () => {}, { virtual: true });
jest.mock("~icons/mdi-light/chevron-double-down", () => {}, { virtual: true });
jest.mock("~icons/mdi-light/chevron-left", () => {}, { virtual: true });
jest.mock("~icons/mdi-light/chevron-right", () => {}, { virtual: true });
jest.mock("~icons/mdi-light/magnify", () => {}, { virtual: true });
jest.mock("~icons/cil/columns", () => {}, { virtual: true });
jest.mock("~icons/octicon/container-24", () => {}, { virtual: true });
jest.mock("~icons/mdi-light/cog", () => {}, { virtual: true });

const localVue = createLocalVue();

localVue.use(Vuex);

describe("<App />", () => {
  const stubs = { RouterLink: RouterLinkStub, "router-view": true, "chevron-left-icon": true };
  let store;

  beforeEach(() => {
    global.EventSource = EventSource;
    const state = {
      settings: { menuWidth: 15 },
      containers: [{ id: "abc", name: "Test 1" }],
    };

    const getters = {
      visibleContainers(store) {
        return store.containers;
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
    wrapper.vm.$store.state.containers = [
      { id: "abc", name: "Test 1" },
      { id: "xyz", name: "Test 2" },
    ];
    await wrapper.vm.$nextTick();

    expect(wrapper.vm.title).toContain("2 containers");
  });

  test("renders correctly", async () => {
    const wrapper = shallowMount(App, { stubs, store, localVue });
    await wrapper.vm.$nextTick();
    expect(wrapper.element).toMatchSnapshot();
  });
});
