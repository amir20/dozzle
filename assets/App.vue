<template>
  <main>
    <menu-mobile v-if="isMobile"></menu-mobile>

    <splitpanes @resized="onResized($event)">
      <pane min-size="10" :size="settings.menuWidth" v-if="!isMobile && !collapseNav">
        <menu-side></menu-side>
      </pane>
      <pane min-size="10">
        <splitpanes>
          <pane class="has-min-height router-view">
            <search></search>
            <router-view></router-view>
          </pane>
          <pane v-for="other in activeContainers" :key="other.id" v-if="!isMobile">
            <log-container
              :id="other.id"
              show-title
              scrollable
              closable
              @close="removeActiveContainer(other)"
            ></log-container>
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
        <icon :name="collapseNav ? 'chevron-right' : 'chevron-left'"></icon>
      </span>
    </button>
  </main>
</template>

<script>
import { mapActions, mapGetters, mapState } from "vuex";
import { Splitpanes, Pane } from "splitpanes";

import LogContainer from "./components/LogContainer";
import MenuSide from "./components/MenuSide";
import MenuMobile from "./components/MenuMobile";
import Search from "./components/Search";
import Icon from "./components/Icon";

export default {
  name: "App",
  components: {
    Icon,
    MenuSide,
    LogContainer,
    MenuMobile,
    Splitpanes,
    Pane,
    Search,
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
  mounted() {
    if (this.hasSmallerScrollbars) {
      document.documentElement.classList.add("has-custom-scrollbars");
    }
    if (this.hasLightTheme) {
      document.documentElement.setAttribute("data-theme", "light");
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
    hasLightTheme(newValue, oldValue) {
      if (newValue) {
        document.documentElement.setAttribute("data-theme", "light");
      } else {
        document.documentElement.removeAttribute("data-theme");
      }
    },
    visibleContainers() {
      this.title = `${this.visibleContainers.length} containers`;
    },
  },
  computed: {
    ...mapState(["isMobile", "settings"]),
    ...mapGetters(["visibleContainers", "activeContainers"]),
    hasSmallerScrollbars() {
      return this.settings.smallerScrollbars;
    },
    hasLightTheme() {
      return this.settings.lightTheme;
    },
  },
  methods: {
    ...mapActions({
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
::v-deep .splitpanes--vertical > .splitpanes__splitter {
  min-width: 3px;
  background: var(--border-color);
  &:hover {
    background: var(--border-hover-color);
  }
}

@media screen and (max-width: 768px) {
  .router-view {
    padding-top: 75px;
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
