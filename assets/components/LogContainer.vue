<template>
  <scrollable-view :scrollable="scrollable" v-if="container">
    <template v-slot:header v-if="showTitle">
      <div class="columns is-vcentered pr-4 mr-0">
        <div class="column is-narrow">
          <container-title :value="container.name" @close="$emit('close')"></container-title>
        </div>
        <div class="column is-clipped">
          <container-stat :stat="container.stat" :state="container.state"></container-stat>
        </div>
        <div class="column is-narrow" v-if="closable">
          <button class="delete is-medium" @click="$emit('close')"></button>
        </div>
      </div>
    </template>
    <template v-slot="{ setLoading }">
      <log-viewer-with-source :id="id" @loading-more="setLoading($event)"></log-viewer-with-source>
    </template>
  </scrollable-view>
</template>

<script>
import { mapGetters } from "vuex";

import LogViewerWithSource from "./LogViewerWithSource";
import ScrollableView from "./ScrollableView";
import ContainerTitle from "./ContainerTitle";
import ContainerStat from "./ContainerStat";

export default {
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
    ScrollableView,
    ContainerTitle,
    ContainerStat,
  },
  computed: {
    ...mapGetters(["allContainersById"]),
    container() {
      return this.allContainersById[this.id];
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
