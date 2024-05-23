<template>
  <ScrollableView :scrollable="scrollable" v-if="container">
    <template #header v-if="showTitle">
      <div class="mx-2 flex items-center gap-2 md:ml-4">
        <ContainerTitle :container="container" />
        <MultiContainerStat class="ml-auto" :containers="[container]" />

        <ContainerActionsToolbar @clear="viewer?.clear()" class="mobile-hidden" :container="container" />
        <a class="btn btn-circle btn-xs" @click="close()" v-if="closable">
          <mdi:close />
        </a>
      </div>
    </template>
    <template #default="{ setLoading }">
      <ViewerWithSource
        ref="viewer"
        @loading-more="setLoading($event)"
        :stream-source="useContainerStream"
        :entity="container"
        :visible-keys="visibleKeys"
        :show-container-name="false"
      />
    </template>
  </ScrollableView>
</template>

<script lang="ts" setup>
import ViewerWithSource from "@/components/LogViewer/ViewerWithSource.vue";
import { ComponentExposed } from "vue-component-type-helpers";

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

const viewer = ref<ComponentExposed<typeof ViewerWithSource>>();

provideLoggingContext(toRef(() => [container.value]));
</script>
