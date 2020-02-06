<template lang="html">
  <main>
    <mobile-menu v-if="isMobile"></mobile-menu>
    <splitpanes @resized="onResized($event)" @resize="onResize($event)">
      <pane :size="settings.menuWidth" v-if="!isMobile" class="menu-pane">
        <side-menu v-show="menuWidth > 10"></side-menu>
        <button
          @click="updateMenuWidth(20)"
          class="button is-small is-primary is-rounded is-inverted"
          id="hide-nav"
          v-show="menuWidth == 0"
        >
          <span class="icon">
            <ion-icon name="arrow-dropright" size="large"></ion-icon>
          </span>
        </button>
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
    ContainerTitle
  },
  data() {
    return {
      title: "",
      menuWidth: 20
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
    }
  },
  computed: {
    ...mapState(["containers", "activeContainers", "isMobile", "settings"]),
    hasSmallerScrollbars() {
      return this.settings.smallerScrollbars;
    }
  },
  methods: {
    ...mapActions({
      fetchContainerList: "FETCH_CONTAINERS",
      removeActiveContainer: "REMOVE_ACTIVE_CONTAINER",
      updateSetting: "UPDATE_SETTING"
    }),
    updateMenuWidth(menuWidth) {
      this.$nextTick(() => (this.menuWidth = menuWidth));
      this.$nextTick(() => this.updateSetting({ menuWidth }));
    },
    onResized(e) {
      if (e.length == 2) {
        this.updateMenuWidth(e[0].size);
      }
    },
    onResize(e) {
      if (e.length == 2) {
        this.menuWidth = e[0].size;
      }
    }
  }
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
  left: -35px;
  bottom: 50%;
  background: black;
  width: 60px;
  color: white;
  border-color: rgb(255, 221, 87);
  padding-left: 40px;
  &:hover {
    left: -25px;
  }
}
</style>
