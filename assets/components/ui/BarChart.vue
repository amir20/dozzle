<template>
  <div
    ref="chartContainer"
    class="relative flex touch-pan-y items-end gap-[2px]"
    @mousemove="onContainerHover"
    @mouseleave="onLeave"
    @touchstart="onContainerTouch"
    @touchmove="onContainerTouch"
  >
    <!-- Inside the root element rather than above it: a comment above compiles to
         a second root node in dev builds, and `wrapper.element` is then the
         comment, so every `trigger()` in the spec fires at nothing. Vue itself
         copes (attrs still fall through to the single element child).

         `touch-pan-y` keeps a vertical swipe scrolling the panel while a
         horizontal drag scrubs the bars, so the chart reads out on a phone
         without trapping the scroll. -->
    <div
      v-for="(bar, i) in downsampledBars"
      :key="i"
      class="bar min-h-px flex-1 rounded-t-sm transition-opacity"
      :class="[barClass, bar.sampled ? 'opacity-70 hover:opacity-100' : 'opacity-15']"
      :style="{ '--height': `${maxValue > 0 ? (bar.percent / maxValue) * 100 : 0}%` }"
    ></div>

    <!-- A 1px guide on the hovered column. The bar's own opacity shift is invisible
         at 3px wide and at `h-4` in a table row, so without this the readout above
         changes with nothing on the chart saying which bar it belongs to. Drawn
         over the bars rather than behind them, which needs no stacking context on
         the ~100 bars themselves. -->
    <div
      v-if="guideLeft !== null"
      class="bg-base-content/25 pointer-events-none absolute inset-y-0 w-px"
      :style="{ left: `${guideLeft}px` }"
    ></div>
  </div>
</template>

<style scoped>
.bar {
  height: var(--height);
  will-change: height;
  contain: layout;
}
</style>

<script setup lang="ts">
export interface BarDataPoint {
  percent: number;
  value: number;
}

// A drawn bar. `sampled` is false for the padding in front of a series that has
// not filled its window yet: it is drawn as a faint track, never averaged, and
// never reported to the hover readout.
type Bar = BarDataPoint & { sampled: boolean };

const {
  chartData,
  barClass = "",
  max,
  sampledFrom = 0,
} = defineProps<{
  chartData: BarDataPoint[];
  barClass?: string;
  // Index in `chartData` where real samples begin. Everything before it is padding
  // that keeps the chart full width while the series scrolls in (see
  // `Container.statsHistory`), and is not a measurement. Defaults to 0 for a series
  // that is real all the way back, such as the cloud history panel.
  sampledFrom?: number;
  // A fixed ceiling for `percent`. Without one the chart scales to its own peak,
  // which suits a spiky series like CPU but draws a steady one (memory) as a solid
  // block at 80% height no matter how little of the capacity it uses.
  max?: number;
}>();

// The bucket's index and how many there are ride along with the value: a
// history chart wants to say when, and only the parent knows what the bars were
// downsampled from. Consumers that just want the number ignore them.
const hoverValue = defineEmit<[value: number, index: number, bars: number]>();
// The chart is the thing that knows the pointer left it, so it says so rather
// than leaving every consumer to hang a @mouseleave on whichever ancestor it
// happens to own.
const hoverEnd = defineEmit<[]>();

const chartContainer = ref<HTMLElement | null>(null);
const { width } = useElementSize(chartContainer);

const BAR_WIDTH = 3;
const GAP = 2;

const availableBars = computed(() => Math.floor(width.value / (BAR_WIDTH + GAP)));
const bucketSize = computed(() => Math.ceil(chartData.length / availableBars.value));

const downsampledBars = ref<Bar[]>([]);
const hoverIndex = ref<number | null>(null);
const maxValue = computed(() => {
  if (max !== undefined) return max;
  const dataMax = Math.max(0, ...downsampledBars.value.map((b) => b.percent));
  return Math.max(dataMax * 1.25, 1);
});
// Full recalculate when width/bucket size changes
watch([availableBars, bucketSize], () => {
  recalculate();
  changeCounter.value = 0;
});

// On data changes, only update the last bar unless a new bucket boundary is crossed.
// A wholesale replacement of the series (e.g. switching containers) is not detected
// here; the parent owns that and must call the exposed recalculate() on switch.
const changeCounter = ref(0);
let initialized = false;
watch(
  () => chartData.at(-1),
  () => {
    if (!initialized) {
      initialized = true;
      recalculate();
      return;
    }
    changeCounter.value++;
    if (changeCounter.value >= bucketSize.value) {
      recalculate();
      changeCounter.value = 0;
    } else {
      updateLastBar();
    }
  },
);

defineExpose({ recalculate });

function averageBucket(bucket: BarDataPoint[], sampled: boolean): Bar {
  const percent = bucket.reduce((sum, d) => sum + d.percent, 0) / bucket.length;
  const value = bucket.reduce((sum, d) => sum + d.value, 0) / bucket.length;
  return { percent, value, sampled };
}

function recalculate() {
  if (availableBars.value === 0) return;

  if (chartData.length <= availableBars.value) {
    downsampledBars.value = chartData.map((d, i) => ({ ...d, sampled: i >= sampledFrom }));
    return;
  }

  const size = bucketSize.value;
  const result: Bar[] = [];
  const numBuckets = Math.ceil(chartData.length / size);

  for (let i = 0; i < numBuckets; i++) {
    const start = i * size;
    const end = Math.min(start + size, chartData.length);
    // A bucket counts as sampled as soon as it holds one real point, so the
    // boundary never eats a measurement to keep the padding tidy.
    result.push(averageBucket(chartData.slice(start, end), end > sampledFrom));
  }

  downsampledBars.value = result.slice(-availableBars.value);
}

function updateLastBar() {
  if (downsampledBars.value.length === 0) return;

  const size = bucketSize.value;
  const lastBucketStart = (Math.ceil(chartData.length / size) - 1) * size;
  const bucket = chartData.slice(lastBucketStart);

  downsampledBars.value[downsampledBars.value.length - 1] = averageBucket(bucket, chartData.length > sampledFrom);
}

function onContainerHover(event: MouseEvent) {
  emitAt(event.clientX);
}

// A touch screen never fires mousemove, so without this the readout above a
// history chart was stuck on the peak and every bar was a shape with no value.
// The last touched bar stays reported after the finger lifts: there is no
// `mouseleave` to fall back from, and a value that vanishes on lift is one
// nobody can read.
function onContainerTouch(event: TouchEvent) {
  const touch = event.touches[0];
  if (touch) emitAt(touch.clientX);
}

function onLeave() {
  hoverIndex.value = null;
  hoverEnd();
}

// Where to draw the guide, in px from the chart's left edge.
const guideLeft = computed(() => {
  const count = downsampledBars.value.length;
  if (hoverIndex.value === null || count === 0 || width.value === 0) return null;
  return (hoverIndex.value + 0.5) * ((width.value + GAP) / count);
});

function emitAt(clientX: number) {
  if (!chartContainer.value) return;

  const count = downsampledBars.value.length;
  if (count === 0) return;

  // Every bar is a `flex-1` sibling with one gap between each pair, so the columns
  // are uniform by construction and the index is arithmetic: a column is
  // (width + GAP) / count wide. Asking each bar for its own rect walked the whole
  // chart on every pointer move to arrive at the same number.
  const rect = chartContainer.value.getBoundingClientRect();
  const pitch = (rect.width + GAP) / count;
  if (pitch <= 0) return;

  const index = Math.min(count - 1, Math.max(0, Math.floor((clientX - rect.left) / pitch)));

  // Padding is not a measurement. Reporting it would put a value and a timestamp
  // on a sample nobody took, so the pointer reads as though it left the chart.
  if (!downsampledBars.value[index].sampled) {
    onLeave();
    return;
  }

  hoverIndex.value = index;
  hoverValue(downsampledBars.value[index].value, index, count);
}
</script>
