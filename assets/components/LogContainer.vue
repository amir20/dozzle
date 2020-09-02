<template>
  <scrollable-view :scrollable="scrollable">
    <template v-slot:header v-if="showTitle">
      <container-title :value="title" :closable="closable" @close="$emit('close')"></container-title>
      <container-stat :id="id"></container-stat>
    </template>
    <template v-slot="{ setLoading }">
      <log-viewer-with-source :id="id" @loading-more="setLoading($event)"></log-viewer-with-source>
    </template>
  </scrollable-view>
</template>

<script>
import LogViewerWithSource from "./LogViewerWithSource";
import ScrollableView from "./ScrollableView";
import ContainerTitle from "./ContainerTitle";
import ContainerStat from "./ContainerStat";

export default {
  props: {
    id: {
      type: String,
    },
    title: {
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
};
</script>
