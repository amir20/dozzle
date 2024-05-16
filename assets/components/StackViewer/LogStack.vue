<template>
  <scrollable-view>
    <template #header>
      <div class="mx-2 flex items-center gap-2 md:ml-4">
        {{ name }}
      </div>
    </template>
    <template #default="{ setLoading }"> </template>
  </scrollable-view>
</template>

<script lang="ts" setup>
import LogViewerWithSource from "@/components/LogViewer/LogViewerWithSource.vue";

const { name } = defineProps<{
  name: string;
}>();

// const store = useContainerStore();
// const container = store.currentContainer($$(id));
// provideContainerContext(container);

// const viewer = ref<InstanceType<typeof LogViewerWithSource>>();

// function onClearClicked() {
//   viewer.value?.clear();
// }

// onKeyStroke("k", (e) => {
//   if ((e.ctrlKey || e.metaKey) && e.shiftKey) {
//     onClearClicked();
//     e.preventDefault();
//   }
// });
</script>
