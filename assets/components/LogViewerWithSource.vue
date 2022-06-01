<template>
  <log-event-source ref="source" #default="{ messages }" @loading-more="emit('loading-more', $event)">
    <log-viewer :messages="messages"></log-viewer>
  </log-event-source>
</template>

<script lang="ts" setup>
import LogViewer from "./LogViewer.vue";
import { ref } from "vue";

const emit = defineEmits(["loading-more"]);

const source = ref<InstanceType<typeof LogViewer>>();
function clear() {
  source.value?.clear();
}
defineExpose({
  clear,
});
</script>
