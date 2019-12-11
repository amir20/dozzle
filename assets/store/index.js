import Vue from "vue";
import Vuex from "vuex";

const mql = window.matchMedia("(max-width: 770px)");

Vue.use(Vuex);

const state = {
  containers: [],
  activeContainers: [],
  searchFilter: null,
  mobileWidth: mql.matches
};

const mutations = {
  SET_CONTAINERS(state, containers) {
    state.containers = containers;
  },
  ADD_ACTIVE_CONTAINERS(state, container) {
    state.activeContainers.push(container);
  },
  REMOVE_ACTIVE_CONTAINER(state, container) {
    state.activeContainers.splice(state.activeContainers.indexOf(container), 1);
  },
  SET_SEARCH(state, filter) {
    state.searchFilter = filter;
  },
  SET_MOBILE_WIDTH(state, value) {
    state.mobileWidth = value;
  }
};

const actions = {
  APPEND_ACTIVE_CONTAINER({ commit }, container) {
    commit("ADD_ACTIVE_CONTAINERS", container);
  },
  REMOVE_ACTIVE_CONTAINER({ commit }, container) {
    commit("REMOVE_ACTIVE_CONTAINER", container);
  },
  SET_SEARCH({ commit }, filter) {
    commit("SET_SEARCH", filter);
  },
  async FETCH_CONTAINERS({ commit }) {
    const containers = await (await fetch(`${BASE_PATH}/api/containers.json`)).json();
    commit("SET_CONTAINERS", containers);
  }
};
const getters = {};

const es = new EventSource(`${BASE_PATH}/api/events/stream`);
es.addEventListener("containers-changed", e => setTimeout(() => store.dispatch("FETCH_CONTAINERS"), 1000), false);

mql.addListener(e => store.commit("SET_MOBILE_WIDTH", e.matches));

const store = new Vuex.Store({
  strict: true,
  state,
  getters,
  actions,
  mutations
});

export default store;
