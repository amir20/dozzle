<template lang="html">
  <splitpanes class="default-theme">
    <pane>
      <log-event-source :id="id" v-slot="eventSource">
        <log-viewer :messages="eventSource.messages"></log-viewer>
      </log-event-source>
    </pane>
    <pane v-for="other in activeContainers">
      <log-event-source :id="other" v-slot="eventSource">
        <log-viewer :messages="eventSource.messages"></log-viewer>
      </log-event-source>
    </pane>
  </splitpanes>
</template>

<script>
import { mapActions, mapGetters, mapState } from "vuex";
import { Splitpanes, Pane } from "splitpanes";
import LogEventSource from "../components/LogEventSource";
import LogViewer from "../components/LogViewer";

export default {
  props: ["id", "name"],
  name: "Container",
  components: {
    LogViewer,
    LogEventSource,
    Splitpanes,
    Pane
  },
  metaInfo() {
    return {
      title: this.name,
      titleTemplate: "%s - Dozzle"
    };
  },
  computed: {
    ...mapState(["activeContainers"])
  }
};
</script>
<style lang="scss" scoped>
.splitpanes__pane {
  background-color: unset !important;
}

::v-deep .splitpanes__splitter {
  width: 4px !important;
  background-color: #aaa;
  border: unset;
}
</style>
