<template>
  <scrollable-view :scrollable="scrollable" v-if="container">
    <template v-slot:header v-if="showTitle">
      <div class="mr-0 columns is-vcentered is-marginless is-hidden-mobile">
        <div class="column is-clipped is-paddingless">
          <container-title :container="container" @close="$emit('close')"></container-title>
        </div>
        <div class="column is-narrow is-paddingless">
          <container-stat :stat="container.stat" :state="container.state"></container-stat>
        </div>
        <div class="mr-2 column is-narrow is-paddingless" v-if="closable">
          <button class="delete is-medium" @click="$emit('close')"></button>
        </div>
        <log-actions-toolbar :container="container" :onClearClicked="onClearClicked"></log-actions-toolbar>
      </div>
    </template>
    <template v-slot="{ setLoading }">
      <log-viewer-with-source ref="viewer" :id="id" @loading-more="setLoading($event)"></log-viewer-with-source>
    </template>
  </scrollable-view>
</template>

<script lang="ts" setup>
import { ref, toRefs } from "vue";
import useContainer from "../composables/container";

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
const { id } = toRefs(props);
const { container } = useContainer(id);

const viewer = ref<HTMLElement>();

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
