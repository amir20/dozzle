<template>
  <log-event-source ref="source" :id="id" v-slot="eventSource" @loading-more="emit('loading-more', $event)">
    <log-viewer :messages="eventSource.messages"></log-viewer>
  </log-event-source>
</template>

<script lang="ts" setup>
import LogViewer from "./LogViewer.vue";
import { ref } from "vue";
defineProps({
  id: {
    type: String,
    required: true,
  },
});

const emit = defineEmits(["loading-more"]);

const source = ref<InstanceType<typeof LogViewer>>();
function clear() {
  source.value?.clear();
}
defineExpose({
  clear,
});
</script>
