import { createStore } from "vuex";
import storage from "store/dist/store.modern";
import { DEFAULT_SETTINGS, DOZZLE_SETTINGS_KEY } from "./settings";
import config from "./config";

storage.set(DOZZLE_SETTINGS_KEY, { ...DEFAULT_SETTINGS, ...storage.get(DOZZLE_SETTINGS_KEY) });

interface Container {
  id: string;
  name: string;
  state: string;
  stat: ContainerStat;
}

interface ContainerStat {
  cpu: number;
  memory: number;
  memoryUsage: number;
}

type IdToContainer = { [id: string]: Container };

function allContainersById(containers: Container[]): IdToContainer {
  return containers.reduce((map, obj) => {
    map[obj.id] = obj;
    return map;
  }, {} as IdToContainer);
}
const store = createStore({
  state: {
    containers: [] as Container[],
    activeContainerIds: [] as string[],
    searchFilter: null,
    settings: storage.get(DOZZLE_SETTINGS_KEY),
    authorizationNeeded: config.authorizationNeeded,
  },
  mutations: {
    SET_CONTAINERS(state, containers) {
      const containersById = allContainersById(containers);

      containers.forEach((container: Container) => {
        container.stat =
          containersById[container.id] && containersById[container.id].stat
            ? containersById[container.id].stat
            : { memoryUsage: 0, cpu: 0, memory: 0 };
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
    UPDATE_SETTINGS(state, newValues) {
      state.settings = { ...state.settings, ...newValues };
      storage.set(DOZZLE_SETTINGS_KEY, state.settings);
    },
    UPDATE_CONTAINER(_, { container, data }) {
      for (const [key, value] of Object.entries(data)) {
        container[key] = value;
      }
    },
  },
  actions: {
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
  },
  getters: {
    allContainersById({ containers }) {
      return allContainersById(containers);
    },
    visibleContainers({ containers, settings: { showAllContainers } }) {
      const filter = showAllContainers ? () => true : (c: Container) => c.state === "running";
      return containers.filter(filter);
    },
    activeContainers({ activeContainerIds }, { allContainersById }) {
      return activeContainerIds.map((id) => allContainersById[id]);
    },
  },
});

if (!config.authorizationNeeded) {
  const es = new EventSource(`${config.base}/api/events/stream`);
  es.addEventListener("containers-changed", (e) => store.commit("SET_CONTAINERS", JSON.parse(e.data)), false);
  es.addEventListener("container-stat", (e) => store.dispatch("UPDATE_STATS", JSON.parse(e.data)), false);
  es.addEventListener("container-die", (e) => store.dispatch("UPDATE_CONTAINER", JSON.parse(e.data)), false);
}

export default store;
