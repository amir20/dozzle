<template>
  <scrollable-view :scrollable="scrollable" v-if="container">
    <template #header v-if="showTitle">
      <div class="mx-2 flex items-center gap-2">
        <container-title @close="$emit('close')" />
        <container-stat class="ml-auto" />

        <!-- <div class="">
          <log-actions-toolbar @clear="onClearClicked()" />
        </div> -->
        <a class="btn btn-circle btn-outline btn-xs" @click="close()" v-if="closable">
          <mdi:close />
        </a>
      </div>
    </template>
    <template #default="{ setLoading }">
      <log-viewer-with-source ref="viewer" @loading-more="setLoading($event)" />
    </template>
  </scrollable-view>
</template>

<script lang="ts" setup>
import LogViewerWithSource from "./LogViewerWithSource.vue";

const {
  id,
  showTitle = false,
  scrollable = false,
  closable = false,
} = defineProps<{
  id: string;
  showTitle?: boolean;
  scrollable?: boolean;
  closable?: boolean;
}>();

const close = defineEmit();

const store = useContainerStore();

const container = store.currentContainer($$(id));
const config = reactive({ stdout: true, stderr: true });

provide("container", container);
provide("stream-config", config);

const viewer = ref<InstanceType<typeof LogViewerWithSource>>();

function onClearClicked() {
  viewer.value?.clear();
}

onKeyStroke("k", (e) => {
  if ((e.ctrlKey || e.metaKey) && e.shiftKey) {
    onClearClicked();
    e.preventDefault();
  }
});
</script>
