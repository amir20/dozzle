<template lang="html">
  <scrollable-view>
    <log-viewer-with-source :id="id"></log-viewer-with-source>
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
