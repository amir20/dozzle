<template lang="html">
  <div class="is-scrollable">
    <search></search>
    <log-viewer-with-source :id="id"></log-viewer-with-source>
  </div>
</template>

<script>
import LogViewerWithSource from "../components/LogViewerWithSource";
import Search from "../components/Search";

export default {
  props: ["id", "name"],
  name: "Container",
  components: {
    LogViewerWithSource,
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
