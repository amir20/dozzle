<template>
  <ScrollableView :scrollable="scrollable" v-if="container">
    <template #header v-if="showTitle">
      <div class="@container mx-2 flex items-center gap-2 md:ml-4">
        <ContainerTitle :container="container" />

        <!-- This list is pixel-identical to the live one, so without a standing label the
             only clue that the stream is frozen is the timestamp in the URL. Relative
             rather than an absolute stamp: "8 hours ago" says how stale the view is at a
             glance, and the exact instant is a hover away. A range names itself in the
             picker instead. -->
        <span
          v-if="timeRange.kind !== 'range'"
          class="text-base-content/60 flex shrink-0 items-center gap-1.5 text-xs whitespace-nowrap max-sm:hidden"
          :title="exactTime"
        >
          <mdi:history class="size-3.5 shrink-0" />
          {{ $t("label.historical-at", { time: relativeTime }) }}
        </span>

        <div class="ml-auto flex items-center gap-1">
          <button
            v-if="timeRange.kind === 'range'"
            type="button"
            class="btn btn-square btn-ghost btn-xs"
            :title="$t('time-range.shift-earlier')"
            :aria-label="$t('time-range.shift-earlier')"
            @click="shift(-1)"
          >
            <mdi:chevron-left />
          </button>
          <!-- The toolbar is hidden on a phone here, so the chip stays. -->
          <TimeRangeMenu :container="container" :range="timeRange" :anchor="until ? undefined : date" />
          <TimeRangeDialog />
          <button
            v-if="timeRange.kind === 'range'"
            type="button"
            class="btn btn-square btn-ghost btn-xs"
            :title="$t('time-range.shift-later')"
            :aria-label="$t('time-range.shift-later')"
            @click="shift(1)"
          >
            <mdi:chevron-right />
          </button>
        </div>

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
      <!-- Where the lines in the range sit against what came before and after, and
           a drag across it is the quickest way to narrow in on an incident. -->
      <LogVolumeChart
        v-if="timeRange.kind === 'range' && !volumeUnsupported && !(volume && isEmptyHistogram(volume))"
        class="mx-2 mt-2 md:mx-4"
        height-class="h-8"
        :data="volume"
        :selection="{ from: timeRange.from, to: timeRange.until }"
        selectable
        @select="selectRange"
      />
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
import ViewerWithSource from "@/components/logs/ViewerWithSource.vue";
import { HistoricalContainer } from "@/models/Container";
import { ComponentExposed } from "vue-component-type-helpers";
import { timeRangeRoute, type TimeRange } from "@/composable/logs/timeRange";
import { isEmptyHistogram } from "@/composable/logs/logHistogram";

const {
  id,
  showTitle = false,
  scrollable = false,
  closable = false,
  date,
  until,
} = defineProps<{
  id: string;
  showTitle?: boolean;
  scrollable?: boolean;
  closable?: boolean;
  date: Date;
  until?: Date;
}>();

const close = defineEmit();

const store = useContainerStore();
const container = store.currentContainer(toRef(() => id));
const historicalContainer = toRef(() => new HistoricalContainer(container.value, date, until));
// A single moment (an alert's link) is not a range: it downloads and filters as
// the whole log does, and the picker opens around it.
const timeRange = computed<TimeRange>(() => (until ? { kind: "range", from: date, until } : { kind: "live" }));

const router = useRouter();
function selectRange(from: Date, to: Date) {
  router.replace(timeRangeRoute(id, { kind: "range", from, until: to }));
}
// ‹ › move the window by its own length, the way a pager turns a page.
function shift(direction: -1 | 1) {
  if (!until) return;
  const span = (until.getTime() - date.getTime()) * direction;
  selectRange(new Date(date.getTime() + span), new Date(until.getTime() + span));
}

// The chart shows the range with as much again on each side, up to now.
const volumeWindow = computed(() => {
  if (!until) return undefined;
  const span = until.getTime() - date.getTime();
  return {
    from: new Date(date.getTime() - span),
    to: new Date(Math.min(Date.now(), until.getTime() + span)),
  };
});
const { data: volume, unsupported: volumeUnsupported } = useLogHistogram(
  toRef(() => container.value!),
  volumeWindow,
  90,
);
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
  { showContainerName: false, showHostname: false, historical: true, timeRange },
);
</script>
