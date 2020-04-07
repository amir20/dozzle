<template lang="html">
  <main>
    <mobile-menu v-if="isMobile"></mobile-menu>

    <splitpanes @resized="onResized($event)">
      <pane min-size="10" :size="settings.menuWidth" v-if="!isMobile && !collapseNav">
        <side-menu></side-menu>
      </pane>
      <pane min-size="10">
        <splitpanes>
          <pane class="has-min-height">
            <search></search>
            <router-view></router-view>
          </pane>
          <pane v-for="other in activeContainers" :key="other.id" v-if="!isMobile">
            <scrollable-view>
              <template v-slot:header>
                <container-title :value="other.name" closable @close="removeActiveContainer(other)"></container-title>
              </template>
              <log-viewer-with-source :id="other.id"></log-viewer-with-source>
            </scrollable-view>
          </pane>
        </splitpanes>
      </pane>
    </splitpanes>
    <button
      @click="collapseNav = !collapseNav"
      class="button is-small is-rounded is-settings-control"
      :class="{ collapsed: collapseNav }"
      id="hide-nav"
      v-if="!isMobile"
    >
      <span class="icon">
        <ion-icon :name="collapseNav ? 'arrow-dropright' : 'arrow-dropleft'" size="large"></ion-icon>
      </span>
    </button>
  </main>
</template>

<script>
import { mapActions, mapGetters, mapState } from "vuex";
import { Splitpanes, Pane } from "splitpanes";

import LogViewerWithSource from "./components/LogViewerWithSource";
import ScrollableView from "./components/ScrollableView";
import SideMenu from "./components/SideMenu";
import MobileMenu from "./components/MobileMenu";
import Search from "./components/Search";
import ContainerTitle from "./components/ContainerTitle";

export default {
  name: "App",
  components: {
    LogViewerWithSource,
    SideMenu,
    MobileMenu,
    ScrollableView,
    Splitpanes,
    Pane,
    Search,
    ContainerTitle,
  },
  data() {
    return {
      title: "",
      collapseNav: false,
    };
  },
  metaInfo() {
    return {
      title: this.title,
      titleTemplate: "%s - Dozzle",
    };
  },
  async created() {
    await this.fetchContainerList();
    this.title = `${this.containers.length} containers`;
  },
  mounted() {
    if (this.hasSmallerScrollbars) {
      document.documentElement.classList.add("has-custom-scrollbars");
    }
    this.menuWidth = this.settings.menuWidth;
  },
  watch: {
    hasSmallerScrollbars(newValue, oldValue) {
      if (newValue) {
        document.documentElement.classList.add("has-custom-scrollbars");
      } else {
        document.documentElement.classList.remove("has-custom-scrollbars");
      }
    },
  },
  computed: {
    ...mapState(["containers", "activeContainers", "isMobile", "settings"]),
    hasSmallerScrollbars() {
      return this.settings.smallerScrollbars;
    },
  },
  methods: {
    ...mapActions({
      fetchContainerList: "FETCH_CONTAINERS",
      removeActiveContainer: "REMOVE_ACTIVE_CONTAINER",
      updateSetting: "UPDATE_SETTING",
    }),
    onResized(e) {
      if (e.length == 2) {
        const menuWidth = e[0].size;
        this.updateSetting({ menuWidth });
      }
    },
  },
};
</script>

<style scoped lang="scss">
::v-deep .splitpanes__splitter {
  min-width: 4px;
  background: #666;
  &:hover {
    background: rgb(255, 221, 87);
  }
}

.button.has-no-border {
  border-color: transparent !important;
}

.has-min-height {
  min-height: 100vh;
}

#hide-nav {
  position: fixed;
  left: 10px;
  bottom: 10px;
  &.collapsed {
    left: -40px;
    width: 60px;
    padding-left: 40px;
    background: rgba(0, 0, 0, 0.95);

    &:hover {
      left: -25px;
    }
  }
}
</style>
