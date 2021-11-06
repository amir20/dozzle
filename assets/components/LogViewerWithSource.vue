<template>
  <log-event-source ref="logEventSource" :id="id" v-slot="eventSource" @loading-more="$emit('loading-more', $event)">
    <log-viewer :messages="eventSource.messages"></log-viewer>
  </log-event-source>
</template>

<script>
import LogEventSource from "./LogEventSource.vue";
import LogViewer from "./LogViewer.vue";

export default {
  props: ["id"],
  name: "LogViewerWithSource",
  components: {
    LogEventSource,
    LogViewer,
  },
  methods: {
    clear() {
      this.$refs.logEventSource.clear();
    },
  },
};
</script>
