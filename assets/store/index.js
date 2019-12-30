import Vue from "vue";
import Vuex from "vuex";
import storage from "store/dist/store.modern";
import { DEFAULT_SETTINGS, DOZZLE_SETTINGS_KEY } from "./settings";

Vue.use(Vuex);

const mql = window.matchMedia("(max-width: 770px)");

storage.set(DOZZLE_SETTINGS_KEY, { ...DEFAULT_SETTINGS, ...storage.get(DOZZLE_SETTINGS_KEY) });

const state = {
  containers: [],
  activeContainers: [],
  searchFilter: null,
  isMobile: mql.matches,
  settings: storage.get(DOZZLE_SETTINGS_KEY)
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
    state.isMobile = value;
  },
  UPDATE_SETTINGS(state, newValues) {
    state.settings = { ...state.settings, ...newValues };
    storage.set(DOZZLE_SETTINGS_KEY, state.settings);
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
  },
  UPDATE_SETTING({ commit }, setting) {
    commit("UPDATE_SETTINGS", setting);
  }
};
const getters = {};

const es = new EventSource(`${BASE_PATH}/api/events/stream`);
es.addEventListener("containers-changed", e => setTimeout(() => store.dispatch("FETCH_CONTAINERS"), 1000), false);
mql.addListener(e => store.commit("SET_MOBILE_WIDTH", e.matches));

const store = new Vuex.Store({
  state,
  getters,
  actions,
  mutations
});

export default store;
