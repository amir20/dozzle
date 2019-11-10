<template lang="html">
  <log-event-source :id="id" v-slot="eventSource">
    <log-viewer :messages="eventSource.messages"></log-viewer>
  </log-event-source>
</template>

<script>
import LogEventSource from "../components/LogEventSource";
import LogViewer from "../components/LogViewer";

export default {
  props: ["id", "name"],
  name: "Container",
  components: {
    LogViewer,
    LogEventSource
  },
  metaInfo() {
    return {
      title: this.name,
      titleTemplate: "%s - Dozzle"
    };
  },
  methods: {},
  computed: {}
};
</script>
<style scoped></style>
