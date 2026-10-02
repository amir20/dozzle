<template>
  <ScrollableView :scrollable="scrollable" :owns-view-context="!closable" v-if="container">
    <template #header v-if="showTitle">
      <div class="@container mx-2 flex items-center gap-1 md:ml-4 md:gap-2">
        <ContainerTitle :container="container" />
        <MultiContainerStat
          class="ml-auto lg:hidden lg:@3xl:flex"
          :containers="[container]"
          v-if="container.state === 'running'"
        />

        <!-- On a phone the toolbar's menu carries these rows instead. -->
        <TimeRangeMenu v-if="!closable" class="max-md:hidden" :container="container" :range="timeRange" />
        <TimeRangeDialog v-if="!closable" />
        <ContainerActionsToolbar @clear="viewer?.clear()" :container="container" />
        <button
          type="button"
          class="btn btn-circle btn-xs"
          @click="close()"
          v-if="closable"
          :title="$t('toolbar.unpin')"
          :aria-label="$t('toolbar.unpin')"
        >
          <mdi:close />
        </button>
      </div>
    </template>
    <template #default>
      <ViewerWithSource
        ref="viewer"
        :stream-source="useContainerStream"
        :entity="container"
        :visible-keys="visibleKeys"
      />
    </template>
  </ScrollableView>
</template>

<script lang="ts" setup>
import ViewerWithSource from "@/components/logs/ViewerWithSource.vue";
import { ComponentExposed } from "vue-component-type-helpers";
import type { TimeRange } from "@/composable/logs/timeRange";

const {
  id,
  showTitle = false,
  scrollable = false,
  closable = false,
  timeRange = { kind: "live" },
} = defineProps<{
  id: string;
  showTitle?: boolean;
  scrollable?: boolean;
  closable?: boolean;
  timeRange?: TimeRange;
}>();

const close = defineEmit();

const store = useContainerStore();
const container = store.currentContainer(toRef(() => id));
const visibleKeys = persistentVisibleKeysForContainer(container);
const viewer = useTemplateRef<ComponentExposed<typeof ViewerWithSource>>("viewer");

provideLoggingContext(
  toRef(() => [container.value]),
  { showContainerName: false, showHostname: false, timeRange: toRef(() => timeRange) },
);
</script>
