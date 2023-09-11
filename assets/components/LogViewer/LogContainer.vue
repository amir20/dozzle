<template>
  <scrollable-view :scrollable="scrollable" v-if="container">
    <template #header v-if="showTitle">
      <div class="flex items-center">
        <div class="">
          <container-title @close="$emit('close')" />
        </div>
        <div class="ml-auto">
          <container-stat />
        </div>

        <div class="">
          <log-actions-toolbar @clear="onClearClicked()" />
        </div>
        <div class="" v-if="closable">
          <button class="delete is-medium" @click="close()"></button>
        </div>
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
<style lang="scss" scoped>
button.delete {
  background-color: var(--scheme-main-ter);
  opacity: 0.6;

  &:after,
  &:before {
    background-color: var(--text-color);
  }

  &:hover {
    opacity: 1;
  }
}
</style>
