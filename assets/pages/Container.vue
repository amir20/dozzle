<template lang="html">
  <div>
    <search></search>
    <scrollable-view>
      <log-viewer-with-source :id="id"></log-viewer-with-source>
    </scrollable-view>
  </div>
</template>

<script>
import LogViewerWithSource from "../components/LogViewerWithSource";
import Search from "../components/Search";
import ScrollableView from "../components/ScrollableView";

export default {
  props: ["id", "name"],
  name: "Container",
  components: {
    LogViewerWithSource,
    ScrollableView,
    Search
  },
  metaInfo() {
    return {
      title: this.name,
      titleTemplate: "%s - Dozzle"
    };
  }
};
</script>
<style lang="scss" scoped></style>
