<template>
  <div ref="chartContainer" class="flex items-end gap-[2px]" @mousemove="onContainerHover">
    <div
      v-for="(bar, i) in downsampledBars"
      :key="i"
      class="bar min-h-px flex-1 rounded-t-sm"
      :class="barClass"
      :style="{ '--height': `${maxValue > 0 ? (bar.percent / maxValue) * 100 : 0}%` }"
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

const { chartData, barClass = "" } = defineProps<{
  chartData: BarDataPoint[];
  barClass?: string;
}>();

const hoverValue = defineEmit<[value: number]>();

const chartContainer = ref<HTMLElement | null>(null);
const { width } = useElementSize(chartContainer);

const BAR_WIDTH = 3;
const GAP = 2;

const availableBars = computed(() => Math.floor(width.value / (BAR_WIDTH + GAP)));
const bucketSize = computed(() => Math.ceil(chartData.length / availableBars.value));

const downsampledBars = ref<BarDataPoint[]>([]);
const maxValue = computed(() => {
  const dataMax = Math.max(0, ...downsampledBars.value.map((b) => b.percent));
  return Math.min(Math.max(dataMax * 1.25, 1), 100);
});
// Full recalculate when width/bucket size changes
watch([availableBars, bucketSize], () => {
  recalculate();
});

// On data changes, only update the last bar unless a new bucket boundary is crossed
const changeCounter = ref(0);
watch(
  () => chartData.at(-1),
  () => {
    changeCounter.value++;
    if (changeCounter.value >= bucketSize.value) {
      recalculate();
      changeCounter.value = 0;
    } else {
      updateLastBar();
    }
  },
);

function averageBucket(bucket: BarDataPoint[]): BarDataPoint {
  const percent = bucket.reduce((sum, d) => sum + d.percent, 0) / bucket.length;
  const value = bucket.reduce((sum, d) => sum + d.value, 0) / bucket.length;
  return { percent, value };
}

function recalculate() {
  if (chartData.length <= availableBars.value || availableBars.value === 0) {
    downsampledBars.value = [...chartData];
    return;
  }

  const size = bucketSize.value;
  const result: BarDataPoint[] = [];
  const numBuckets = Math.ceil(chartData.length / size);

  for (let i = 0; i < numBuckets; i++) {
    const start = i * size;
    const end = Math.min(start + size, chartData.length);
    result.push(averageBucket(chartData.slice(start, end)));
  }

  downsampledBars.value = result.slice(-availableBars.value);
}

function updateLastBar() {
  if (downsampledBars.value.length === 0) return;

  const size = bucketSize.value;
  const lastBucketStart = (Math.ceil(chartData.length / size) - 1) * size;
  const bucket = chartData.slice(lastBucketStart);

  downsampledBars.value[downsampledBars.value.length - 1] = averageBucket(bucket);
}

function onContainerHover(event: MouseEvent) {
  if (!chartContainer.value) return;

  const bars = chartContainer.value.children;
  if (bars.length === 0) return;

  const mouseX = event.clientX;
  let index = 0;

  // Find the bar whose column contains the mouse x position
  for (let i = 0; i < bars.length; i++) {
    const rect = bars[i].getBoundingClientRect();
    if (mouseX >= rect.left) {
      index = i;
    } else {
      break;
    }
  }

  hoverValue(downsampledBars.value[index].value);
}
</script>
