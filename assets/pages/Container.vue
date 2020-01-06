<template lang="html">
  <scrollable-view>
    <template v-slot:header v-if="activeContainers.length > 0">
      <container-title :value="name"></container-title>
    </template>
    <log-viewer-with-source :id="id"></log-viewer-with-source>
  </scrollable-view>
</template>

<script>
import { mapActions, mapGetters, mapState } from "vuex";

import LogViewerWithSource from "../components/LogViewerWithSource";
import ScrollableView from "../components/ScrollableView";
import ContainerTitle from "../components/ContainerTitle";

export default {
  props: ["id", "name"],
  name: "Container",
  components: {
    LogViewerWithSource,
    ScrollableView,
    ContainerTitle
  },
  metaInfo() {
    return {
      title: this.name,
      titleTemplate: "%s - Dozzle"
    };
  },
  computed: {
    ...mapState(["activeContainers", "isMobile"])
  }
};
</script>
