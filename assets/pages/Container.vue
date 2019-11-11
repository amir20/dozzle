<template lang="html">
  <splitpanes class="default-theme">
    <pane>
      <log-event-source :id="id" v-slot="eventSource">
        <log-viewer :messages="eventSource.messages"></log-viewer>
      </log-event-source>
    </pane>
    <pane>
      <log-event-source :id="id" v-slot="eventSource">
        <log-viewer :messages="eventSource.messages"></log-viewer>
      </log-event-source>
    </pane>
    <pane>
      <log-event-source :id="id" v-slot="eventSource">
        <log-viewer :messages="eventSource.messages"></log-viewer>
      </log-event-source>
    </pane>
  </splitpanes>
</template>

<script>
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
  }
};
</script>
<style scoped>
.splitpanes__pane {
  background-color: unset !important;
}
</style>
