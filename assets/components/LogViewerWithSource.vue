<template>
  <log-event-source ref="logEventSource" :id="id" v-slot="eventSource" @loading-more="$emit('loading-more', $event)">
    <log-viewer :messages="eventSource.messages"></log-viewer>
  </log-event-source>
</template>

<script setup>
import { ref } from "vue";
import LogEventSource from "./LogEventSource.vue";
import LogViewer from "./LogViewer.vue";

const props = defineProps({
  id: String,
});

const logEventSource = ref(null);
function clear() {
  this.$refs.logEventSource.clear();
}
defineExpose({
  clear,
});
</script>
