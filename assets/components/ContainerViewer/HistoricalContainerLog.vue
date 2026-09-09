<template>
  <ScrollableView :scrollable="scrollable" v-if="container">
    <template #header v-if="showTitle">
      <div class="@container mx-2 flex items-center gap-2 md:ml-4">
        <ContainerTitle :container="container" />

        <!-- This list is pixel-identical to the live one, so without a standing label the
             only clue that the stream is frozen is the timestamp in the URL. Relative
             rather than an absolute stamp: "8 hours ago" says how stale the view is at a
             glance, and the exact instant is a hover away. -->
        <span
          class="text-base-content/60 flex shrink-0 items-center gap-1.5 text-xs whitespace-nowrap max-sm:hidden"
          :title="exactTime"
        >
          <mdi:history class="size-3.5 shrink-0" />
          {{ $t("label.historical-at", { time: relativeTime }) }}
        </span>

        <router-link
          :to="{ name: '/container/[id]', params: { id: container.id } }"
          class="btn btn-secondary btn-sm shrink-0"
          :title="$t('tooltip.live-logs')"
          v-if="container.state === 'running'"
        >
          <mdi:lightning-bolt />
          {{ $t("button.live-logs") }}
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

// Same shared tick RelativeTime rides, so the label ages without a timer of its own.
const relativeTime = computed(() => {
  relativeTimeTick.value;
  return toRelativeTime(date, locale.value === "" ? undefined : locale.value);
});
const exactTime = computed(() => date.toLocaleString());
useTemplateRef<ComponentExposed<typeof ViewerWithSource>>("viewer");

provideLoggingContext(
  toRef(() => [container.value]),
  { showContainerName: false, showHostname: false, historical: true },
);
</script>
