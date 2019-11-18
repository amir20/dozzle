<template lang="html">
  <div class="columns is-marginless">
    <aside class="column menu is-3-tablet is-2-widescreen">
      <a
        role="button"
        class="navbar-burger burger is-white is-hidden-tablet is-pulled-right"
        @click="showNav = !showNav"
        :class="{ 'is-active': showNav }"
      >
        <span></span> <span></span> <span></span>
      </a>
      <h1 class="title has-text-warning is-marginless">Dozzle</h1>
      <p class="menu-label is-hidden-mobile" :class="{ 'is-active': showNav }">Containers</p>
      <ul class="menu-list is-hidden-mobile" :class="{ 'is-active': showNav }">
        <li v-for="item in containers">
          <router-link :to="{ name: 'container', params: { id: item.id, name: item.name } }" active-class="is-active">
            <div class="hide-overflow">{{ item.name }}</div>
            <span @click.stop.prevent="appendActiveContainer(item)"><i class="fas fa-thumbtack"></i></span>
          </router-link>
        </li>
      </ul>
    </aside>
    <div class="column is-offset-3-tablet is-offset-2-widescreen is-9-tablet is-10-widescreen is-paddingless">
      <div class="columns is-gapless">
        <div class="column log-container">
          <router-view></router-view>
        </div>
        <div class="column log-container" v-for="other in activeContainers" :key="other.id">
          <div class="name columns is-marginless">
            <span class="column">{{ other.name }}</span>
            <span class="column is-narrow">
              <button class="delete is-medium" @click="removeActiveContainer(other)"></button>
            </span>
          </div>
          <log-event-source :id="other.id" v-slot="eventSource">
            <log-viewer :messages="eventSource.messages"></log-viewer>
          </log-event-source>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { mapActions, mapGetters, mapState } from "vuex";
import { Splitpanes, Pane } from "splitpanes";
import LogEventSource from "./components/LogEventSource";
import LogViewer from "./components/LogViewer";

export default {
  name: "App",
  components: {
    LogViewer,
    LogEventSource,
    Splitpanes,
    Pane
  },
  data() {
    return {
      title: "",
      showNav: false
    };
  },
  metaInfo() {
    return {
      title: this.title,
      titleTemplate: "%s - Dozzle"
    };
  },
  async created() {
    await this.fetchContainerList();
    this.title = `${this.containers.length} containers`;
  },
  computed: {
    ...mapState(["containers", "activeContainers"])
  },
  methods: {
    ...mapActions({
      fetchContainerList: "FETCH_CONTAINERS",
      appendActiveContainer: "APPEND_ACTIVE_CONTAINER",
      removeActiveContainer: "REMOVE_ACTIVE_CONTAINER"
    })
  },
  watch: {
    $route(to, from) {
      this.showNav = false;
    }
  }
};
</script>

<style scoped lang="scss">
.is-hidden-mobile.is-active {
  display: block !important;
}

.navbar-burger {
  height: 2.35rem;
}

aside {
  position: fixed;
  z-index: 2;
  padding: 1em;

  @media screen and (min-width: 769px) {
    & {
      height: 100vh;
      overflow: auto;
    }
  }

  @media screen and (max-width: 768px) {
    & {
      position: sticky;
      top: 0;
      left: 0;
      right: 0;
      background: #222;
    }

    .menu-label {
      margin-top: 1em;
    }
  }
}

.hide-overflow {
  text-overflow: ellipsis;
  white-space: nowrap;
  overflow: hidden;
}

.burger.is-white {
  color: #fff;
}

.log-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background-color: unset !important;

  .log-event-source {
    flex: 1;
    overflow: auto;
  }
}

.name {
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  background: rgba(0, 0, 0, 0.1);
  font-weight: bold;
  font-family: monospace;
}
</style>
