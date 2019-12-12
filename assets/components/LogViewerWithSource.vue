<template lang="html">
  <log-event-source :id="id" v-slot="eventSource" ref="source">
    <log-viewer :messages="eventSource.messages" @fetchMore="$refs.source.fetchMore()"></log-viewer>
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
    LogViewer
  }
};
</script>
