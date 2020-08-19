<template>
  <log-event-source :id="id" v-slot="eventSource" @loading-more="$emit('loading-more', $event)">
    <log-viewer :messages="eventSource.messages"></log-viewer>
  </log-event-source>
</template>

<script>
import LogEventSource from "./LogEventSource";
import LogViewer from "./LogViewer";

export default {
  props: ["id"],
  name: "LogViewerWithSource",
  components: {
    LogEventSource,
    LogViewer,
  },
};
</script>
