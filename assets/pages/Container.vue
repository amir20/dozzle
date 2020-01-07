<template lang="html">
  <scrollable-view :scrollable="activeContainers.length > 0">
    <template v-slot:header v-if="activeContainers.length > 0">
      <container-title :value="allContainersById[id].name"></container-title>
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
  data() {
    return {
      title: "loading"
    };
  },
  metaInfo() {
    return {
      title: this.title
    };
  },
  computed: {
    ...mapState(["activeContainers"]),
    ...mapGetters(["allContainersById"])
  },
  watch: {
    id() {
      this.title = this.allContainersById[this.id].name;
    },
    allContainersById() {
      this.title = this.allContainersById[this.id].name;
    }
  }
};
</script>
