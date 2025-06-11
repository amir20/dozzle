<template>
  <ScrollableView :scrollable="scrollable" v-if="container">
    <template #header v-if="showTitle">
      <div class="@container mx-2 flex items-center gap-2 md:ml-4">
        <ContainerTitle :container="container" />
        <router-link
          :to="{ name: '/container/[id]', params: { id: container.id } }"
          class="btn btn-secondary btn-sm"
          v-if="container.state === 'running'"
        >
          <mdi:lightning-bolt />
          Live Logs
        </router-link>

        <ContainerActionsToolbar class="max-md:hidden" :container="container" historical />
        <a class="btn btn-circle btn-xs" @click="close()" v-if="closable">
          <mdi:close />
        </a>
      </div>
    </template>
    <template #default>
      <ViewerWithSource
        ref="viewer"
        :stream-source="useHistoricalContainerLog"
        :entity="historicalContainer"
        :visible-keys="visibleKeys"
      />
    </template>
  </ScrollableView>
</template>

<script lang="ts" setup>
import ViewerWithSource from "@/components/LogViewer/ViewerWithSource.vue";
import { HistoricalContainer } from "@/models/Container";
import { ComponentExposed } from "vue-component-type-helpers";

const {
  id,
  showTitle = false,
  scrollable = false,
  closable = false,
  date,
} = defineProps<{
  id: string;
  showTitle?: boolean;
  scrollable?: boolean;
  closable?: boolean;
  date: Date;
}>();

const close = defineEmit();

const store = useContainerStore();
const container = store.currentContainer(toRef(() => id));
const historicalContainer = toRef(() => new HistoricalContainer(container.value, date));
const visibleKeys = persistentVisibleKeysForContainer(container);
const viewer = useTemplateRef<ComponentExposed<typeof ViewerWithSource>>("viewer");

provideLoggingContext(
  toRef(() => [container.value]),
  { showContainerName: false, showHostname: false, historical: true },
);
</script>
