<template>
  <log-event-source ref="source" #default="{ messages }" @loading-more="emit('loading-more', $event)">
    <log-viewer :messages="messages"></log-viewer>
  </log-event-source>
</template>

<script lang="ts" setup>
import LogEventSource from "./LogEventSource.vue";

const emit = defineEmits(["loading-more"]);

const source = $ref<InstanceType<typeof LogEventSource>>();
function clear() {
  source?.clear();
}
defineExpose({
  clear,
});
</script>
