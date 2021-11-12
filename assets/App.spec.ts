import EventSource from "eventsourcemock";
import { shallowMount, RouterLinkStub } from "@vue/test-utils";
import { createStore, Store } from "vuex";
import App from "./App.vue";

jest.mock("./store/config.ts", () => ({ base: "" }));

jest.mock("~icons/octicon/download-24", () => {}, { virtual: true });
jest.mock("~icons/octicon/trash-24", () => {}, { virtual: true });
jest.mock("~icons/mdi-light/chevron-double-down", () => {}, { virtual: true });
jest.mock("~icons/mdi-light/chevron-left", () => {}, { virtual: true });
jest.mock("~icons/mdi-light/chevron-right", () => {}, { virtual: true });
jest.mock("~icons/mdi-light/magnify", () => {}, { virtual: true });
jest.mock("~icons/cil/columns", () => {}, { virtual: true });
jest.mock("~icons/octicon/container-24", () => {}, { virtual: true });
jest.mock("~icons/mdi-light/cog", () => {}, { virtual: true });

describe("<App />", () => {
  const stubs = {
    RouterLink: RouterLinkStub,
    "router-view": true,
    "chevron-left-icon": true,
    "chevron-right-icon": true,
  };

  let store: Store<State>;

  interface State {
    settings: { menuWidth: number };
    containers: { id: string; name: string }[];
  }

  beforeEach(() => {
    global.EventSource = EventSource;
    const state = {
      settings: { menuWidth: 15 },
      containers: [{ id: "abc", name: "Test 1" }],
    };

    const getters = {
      visibleContainers(store: State): { id: string; name: string }[] {
        return store.containers;
      },
      activeContainers() {
        return [];
      },
    };

    store = createStore({ state, getters });
  });

  test("has right title", async () => {
    const wrapper = shallowMount(App, {
      global: {
        plugins: [store],
        stubs,
      },
    });
    wrapper.vm.$store.state.containers = [
      { id: "abc", name: "Test 1" },
      { id: "xyz", name: "Test 2" },
    ];
    await wrapper.vm.$nextTick();

    expect(wrapper.vm.title).toContain("2 containers");
  });

  test("renders correctly", async () => {
    const wrapper = shallowMount(App, {
      global: {
        plugins: [store],
        stubs,
      },
    });
    await wrapper.vm.$nextTick();
    expect(wrapper.html()).toMatchSnapshot();
  });
});
