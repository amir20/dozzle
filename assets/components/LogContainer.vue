<template>
  <scrollable-view :scrollable="scrollable" v-if="container">
    <template #header v-if="showTitle">
      <div class="mr-0 columns is-vcentered is-marginless is-hidden-mobile">
        <div class="column is-clipped is-paddingless">
          <container-title @close="$emit('close')" />
        </div>
        <div class="column is-narrow is-paddingless">
          <container-stat v-if="container.stat" />
        </div>

        <div class="mr-2 column is-narrow is-paddingless">
          <log-actions-toolbar :onClearClicked="onClearClicked" />
        </div>
        <div class="mr-2 column is-narrow is-paddingless" v-if="closable">
          <button class="delete is-medium" @click="emit('close')"></button>
        </div>
      </div>
    </template>
    <template #default="{ setLoading }">
      <log-viewer-with-source ref="viewer" @loading-more="setLoading($event)" />
    </template>
  </scrollable-view>
</template>

<script lang="ts" setup>
import { provide, ref, toRefs } from "vue";
import LogViewerWithSource from "./LogViewerWithSource.vue";
import { useContainerStore } from "@/stores/container";

const props = defineProps({
  id: {
    type: String,
    required: true,
  },
  showTitle: {
    type: Boolean,
    default: false,
  },
  scrollable: {
    type: Boolean,
    default: false,
  },
  closable: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(["close"]);

const { id } = toRefs(props);
const store = useContainerStore();

const container = store.currentContainer(id);

provide("container", container);

const viewer = ref<InstanceType<typeof LogViewerWithSource>>();

function onClearClicked() {
  viewer.value?.clear();
}
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
