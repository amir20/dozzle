<template>
  <log-event-source ref="source" #default="{ messages }" @loading-more="loadingMore($event)">
    <log-viewer :messages="messages"></log-viewer>
  </log-event-source>
</template>

<script lang="ts" setup>
import LogEventSource from "./LogEventSource.vue";

const loadingMore = defineEmit<[value: boolean]>();
const source = $ref<InstanceType<typeof LogEventSource>>();
function clear() {
  source?.clear();
}
defineExpose({
  clear,
});
</script>
