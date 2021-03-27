<template>
  <main>
    <b-modal v-model="showFuzzySearch" animation="false">
      <div>
        <div class="panel">
          <div class="panel-block">
            <p class="control has-icons-left">
              <input class="input" type="text" placeholder="Search Containers" ref="search" />
              <span class="icon is-left">
                <icon name="search"></icon>
              </span>
            </p>
          </div>
          <router-link
            :to="{ name: 'container', params: { id: item.id, name: item.name } }"
            v-for="item in containers.slice(0, 10)"
            :key="item.id"
            class="panel-block"
          >
            <span class="name">{{ item.name }}</span>
            <div class="subtitle is-7 status">
              <past-time :date="new Date(item.created * 1000)"></past-time>
            </div>
          </router-link>
        </div>
      </div>
    </b-modal>
    <mobile-menu v-if="isMobile"></mobile-menu>

    <splitpanes @resized="onResized($event)">
      <pane min-size="10" :size="settings.menuWidth" v-if="!isMobile && !collapseNav">
        <side-menu></side-menu>
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

import hotkeys from "hotkeys-js";

import LogContainer from "./components/LogContainer";
import SideMenu from "./components/SideMenu";
import MobileMenu from "./components/MobileMenu";
import Search from "./components/Search";
import PastTime from "./components/PastTime";
import Icon from "./components/Icon";

export default {
  name: "App",
  components: {
    Icon,
    SideMenu,
    LogContainer,
    MobileMenu,
    Splitpanes,
    PastTime,
    Pane,
    Search,
  },
  data() {
    return {
      title: "",
      collapseNav: false,
      showFuzzySearch: false,
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
    hotkeys("command+k, ctrl+k", (event, handler) => {
      this.showFuzzySearch = true;
      setTimeout(() => this.$refs.search.focus(), 0);
    });
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
    ...mapState(["isMobile", "settings", "containers"]),
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

.icon {
  padding: 10px 3px;
}
</style>
