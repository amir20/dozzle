<template lang="html">
  <main>
    <mobile-menu v-if="isMobile"></mobile-menu>
    <splitpanes @resized="onResize($event)">
      <pane min-size="10" :size="settings.menuWidth" v-if="!isMobile">
        <side-menu></side-menu>
      </pane>
      <pane :size="isMobile ? 100 : 100 - settings.menuWidth" min-size="10">
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
    ...mapState(["containers", "activeContainers", "isMobile", "settings"])
  },
  methods: {
    ...mapActions({
      fetchContainerList: "FETCH_CONTAINERS",
      removeActiveContainer: "REMOVE_ACTIVE_CONTAINER",
      updateSetting: "UPDATE_SETTING"
    }),
    onResize(e) {
      if (e.length == 2) {
        this.updateSetting({ menuWidth: Math.min(90, e[0].size) });
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
</style>
