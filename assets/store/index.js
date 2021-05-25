import Vue from "vue";
import Vuex from "vuex";
import storage from "store/dist/store.modern";
import { DEFAULT_SETTINGS, DOZZLE_SETTINGS_KEY } from "./settings";
import config from "./config";

Vue.use(Vuex);

const mql = window.matchMedia("(max-width: 770px)");

storage.set(DOZZLE_SETTINGS_KEY, { ...DEFAULT_SETTINGS, ...storage.get(DOZZLE_SETTINGS_KEY) });

const state = {
  containers: [],
  activeContainerIds: [],
  searchFilter: null,
  isMobile: mql.matches,
  settings: storage.get(DOZZLE_SETTINGS_KEY),
  authorizationNeeded: config.authorizationNeeded,
};

const mutations = {
  SET_CONTAINERS(state, containers) {
    const containersById = getters.allContainersById({ containers });

    containers.forEach((container) => {
      container.stat =
        containersById[container.id] && containersById[container.id].stat
          ? containersById[container.id].stat
          : { memoryUsage: 0, cpu: 0 };
    });

    state.containers = containers;
  },
  ADD_ACTIVE_CONTAINERS(state, { id }) {
    state.activeContainerIds.push(id);
  },
  REMOVE_ACTIVE_CONTAINER(state, { id }) {
    state.activeContainerIds.splice(state.activeContainerIds.indexOf(id), 1);
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
  },
  UPDATE_CONTAINER(_, { container, data }) {
    for (const [key, value] of Object.entries(data)) {
      Vue.set(container, key, value);
    }
  },
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
  UPDATE_SETTING({ commit }, setting) {
    commit("UPDATE_SETTINGS", setting);
  },
  UPDATE_STATS({ commit, getters: { allContainersById } }, stat) {
    const container = allContainersById[stat.id];
    if (container) {
      commit("UPDATE_CONTAINER", { container, data: { stat } });
    }
  },
  UPDATE_CONTAINER({ commit, getters: { allContainersById } }, event) {
    switch (event.name) {
      case "die":
        const container = allContainersById[event.actorId];
        commit("UPDATE_CONTAINER", { container, data: { state: "exited" } });
        break;
      default:
    }
  },
};

const getters = {
  allContainersById({ containers }) {
    return containers.reduce((map, obj) => {
      map[obj.id] = obj;
      return map;
    }, {});
  },
  visibleContainers({ containers, settings: { showAllContainers } }) {
    const filter = showAllContainers ? () => true : (c) => c.state === "running";
    return containers.filter(filter);
  },
  activeContainers({ activeContainerIds }, { allContainersById }) {
    return activeContainerIds.map((id) => allContainersById[id]);
  },
};

if (!config.authorizationNeeded) {
  const es = new EventSource(`${config.base}/api/events/stream`);
  es.addEventListener("containers-changed", (e) => store.commit("SET_CONTAINERS", JSON.parse(e.data)), false);
  es.addEventListener("container-stat", (e) => store.dispatch("UPDATE_STATS", JSON.parse(e.data)), false);
  es.addEventListener("container-die", (e) => store.dispatch("UPDATE_CONTAINER", JSON.parse(e.data)), false);
}

mql.addEventListener("change", (e) => store.commit("SET_MOBILE_WIDTH", e.matches));

const store = new Vuex.Store({
  state,
  getters,
  actions,
  mutations,
});

export default store;
