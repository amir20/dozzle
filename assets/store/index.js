import Vue from "vue";
import Vuex from "vuex";
// import storage from "store/dist/store.modern";

Vue.use(Vuex);

const state = {
  containers: [],
  activeContainers: []
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
  }
};

const actions = {
  APPEND_ACTIVE_CONTAINER({ commit }, container) {
    commit("ADD_ACTIVE_CONTAINERS", container);
  },
  REMOVE_ACTIVE_CONTAINER({ commit }, container) {
    commit("REMOVE_ACTIVE_CONTAINER", container);
  },
  async FETCH_CONTAINERS({ commit }) {
    const containers = await (await fetch(`${BASE_PATH}/api/containers.json`)).json();
    commit("SET_CONTAINERS", containers);
  }
};
const getters = {};

const es = new EventSource(`${BASE_PATH}/api/events/stream`);
es.addEventListener("containers-changed", e => setTimeout(() => store.dispatch("FETCH_CONTAINERS"), 1000), false);

const store = new Vuex.Store({
  strict: true,
  state,
  getters,
  actions,
  mutations
});

export default store;
