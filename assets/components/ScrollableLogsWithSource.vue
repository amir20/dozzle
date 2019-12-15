<template lang="html">
  <scrollable-view @scrolledToTop="$refs.viewer.loadMoreLogs()">
    <log-viewer-with-source :id="id" ref="viewer"></log-viewer-with-source>
  </scrollable-view>
</template>

<script>
import ScrollableView from "./ScrollableView";
import LogViewerWithSource from "./LogViewerWithSource";

export default {
  props: ["id"],
  name: "ScrollableLogsWithSource",
  components: {
    LogViewerWithSource,
    ScrollableView
  }
};
</script>
