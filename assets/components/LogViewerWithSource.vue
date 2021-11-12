<template>
  <log-event-source ref="source" :id="id" v-slot="eventSource" @loading-more="$emit('loading-more', $event)">
    <log-viewer :messages="eventSource.messages"></log-viewer>
  </log-event-source>
</template>

<script lang="ts" setup>
defineProps({
  id: String,
});

const source = ref(null);
function clear() {
  source.value.clear();
}
defineExpose({
  clear,
});
</script>
