<template>
  <scrollable-view :scrollable="scrollable" v-if="container">
    <template v-slot:header v-if="showTitle">
      <div class="mr-0 columns is-vcentered is-marginless is-hidden-mobile">
        <div class="column is-clipped is-paddingless">
          <container-title :container="container" @close="$emit('close')"></container-title>
        </div>
        <div class="column is-narrow is-paddingless">
          <container-stat :stat="container.stat" :state="container.state"></container-stat>
        </div>
        <div class="column is-narrow is-paddingless mr-2" v-if="closable">
          <button class="delete is-medium" @click="$emit('close')"></button>
        </div>
        <!-- <div class="column is-narrow is-paddingless mr-2">
          <o-dropdown aria-role="list" position="bottom-left">
            <template v-slot:trigger>
              <span class="btn">
                <span class="icon">
                  <menu-icon />
                </span>
              </span>
            </template>

            <o-dropdown-item aria-role="listitem"> Clear </o-dropdown-item>
            <o-dropdown-item aria-role="listitem">Download</o-dropdown-item>
          </o-dropdown>
        </div> -->
      </div>
      <log-actions-toolbar :container="container" :onClearClicked="onClearClicked"></log-actions-toolbar>
    </template>
    <template v-slot="{ setLoading }">
      <log-viewer-with-source ref="logViewer" :id="id" @loading-more="setLoading($event)"></log-viewer-with-source>
    </template>
  </scrollable-view>
</template>

<script>
import LogViewerWithSource from "./LogViewerWithSource.vue";
import LogActionsToolbar from "./LogActionsToolbar.vue";
import ScrollableView from "./ScrollableView.vue";
import ContainerTitle from "./ContainerTitle.vue";
import ContainerStat from "./ContainerStat.vue";
import MenuIcon from "~icons/carbon/overflow-menu-vertical";
import containerMixin from "./mixins/container";

export default {
  mixins: [containerMixin],
  props: {
    id: {
      type: String,
    },
    showTitle: {
      type: Boolean,
      default: false,
    },
    scrollable: {
      type: Boolean,
      default: false,
    },
    closable: {
      type: Boolean,
      default: false,
    },
  },
  name: "LogContainer",
  components: {
    LogViewerWithSource,
    LogActionsToolbar,
    ScrollableView,
    ContainerTitle,
    MenuIcon,
    ContainerStat,
  },
  methods: {
    onClearClicked() {
      this.$refs.logViewer.clear();
    },
  },
};
</script>
<style lang="scss" scoped>
button.delete {
  background-color: var(--scheme-main-ter);
  opacity: 0.6;
  &:after,
  &:before {
    background-color: var(--text-color);
  }

  &:hover {
    opacity: 1;
  }
}
</style>
