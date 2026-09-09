<template>
  <EventSource ref="source" #default="{ messages }" :stream-source="streamSource" :entity="entity">
    <LogViewer :messages="messages" :visible-keys="visibleKeys" />
  </EventSource>
</template>

<script lang="ts" setup generic="T">
import EventSource from "@/components/LogViewer/EventSource.vue";
import { LogStreamSource } from "@/composable/eventStreams";
import type { VisibleKeysSource } from "@/composable/visible";
import { ComponentExposed } from "vue-component-type-helpers";

const { streamSource, visibleKeys, entity } = defineProps<{
  streamSource: (t: Ref<T>) => LogStreamSource;
  visibleKeys: VisibleKeysSource;
  entity: T;
}>();

const source = useTemplateRef<ComponentExposed<typeof EventSource>>("source");

defineExpose({
  clear: () => source.value?.clear(),
});

onKeyStroke(["l", "L"], (e) => {
  if ((e.ctrlKey || e.metaKey) && e.shiftKey) {
    source.value?.clear();
    e.preventDefault();
  }
});
</script>
