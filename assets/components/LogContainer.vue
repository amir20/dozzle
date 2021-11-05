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
        <div class="column is-narrow is-paddingless" v-if="closable">
          <button class="delete is-medium" @click="$emit('close')"></button>
        </div>
      </div>
      <log-actions-toolbar :container="container" :onClearClicked="onClearClicked"></log-actions-toolbar>
    </template>
    <template v-slot="{ setLoading }">
      <log-viewer-with-source ref="logViewer" :id="id" @loading-more="setLoading($event)"></log-viewer-with-source>
    </template>
  </scrollable-view>
</template>

<script>
import LogViewerWithSource from "./LogViewerWithSource";
import LogActionsToolbar from "./LogActionsToolbar";
import ScrollableView from "./ScrollableView";
import ContainerTitle from "./ContainerTitle";
import ContainerStat from "./ContainerStat";
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
