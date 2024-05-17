<template>
  <ScrollableView :scrollable="scrollable" v-if="container">
    <template #header v-if="showTitle">
      <div class="mx-2 flex items-center gap-2 md:ml-4">
        <ContainerTitle />
        <ContainerStat class="ml-auto" />

        <ContainerActionsToolbar @clear="onClearClicked()" class="mobile-hidden" />
        <a class="btn btn-circle btn-xs" @click="close()" v-if="closable">
          <mdi:close />
        </a>
      </div>
    </template>
    <template #default="{ setLoading }">
      <ViewerWithSource
        ref="viewer"
        @loading-more="setLoading($event)"
        :stream-source="useContainerContextLogStream"
        :visible-keys="visibleKeys"
      />
    </template>
  </ScrollableView>
</template>

<script lang="ts" setup>
import LogViewerWithSource from "@/components/LogViewer/LogViewerWithSource.vue";

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

const visibleKeys = persistentVisibleKeysForContainer(container);
provideContainerContext(container);

const viewer = ref<InstanceType<typeof LogViewerWithSource>>();

const onClearClicked = () => viewer.value?.clear();

onKeyStroke("k", (e) => {
  if ((e.ctrlKey || e.metaKey) && e.shiftKey) {
    onClearClicked();
    e.preventDefault();
  }
});
</script>
