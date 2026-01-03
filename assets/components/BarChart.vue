<template>
  <div ref="chartContainer" class="flex items-end gap-[2px]">
    <div
      v-for="(dataPoint, i) in downsampledData"
      :key="i"
      class="min-h-px flex-1 rounded-t-sm"
      :class="barClass"
      :style="`height: ${Math.min(dataPoint, 100)}%`"
      @mousemove="onBarHover(i)"
    ></div>
  </div>
</template>

<script setup lang="ts">
const { chartData, barClass = "" } = defineProps<{
  chartData: number[];
  barClass?: string;
}>();

const hoverIndex = defineEmit<[index: number]>();

const chartContainer = ref<HTMLElement | null>(null);
const { width } = useElementSize(chartContainer);

const BAR_WIDTH = 3;
const GAP = 2;

const availableBars = computed(() => Math.floor(width.value / (BAR_WIDTH + GAP)));
const bucketSize = computed(() => Math.ceil(chartData.length / availableBars.value));

const downsampledData = ref<number[]>([]);
const changeCounter = ref(-1);

// Watch chartData changes
watch(
  () => chartData,
  () => {
    // If changeCounter is -1, it means this is the first time the data is loaded
    if (changeCounter.value === -1) {
      recalculate();
    }
    changeCounter.value++;
    if (changeCounter.value >= bucketSize.value) {
      // Recalculate when counter reaches bucket size
      recalculate();
      changeCounter.value = 0;
    }
  },
  { deep: true },
);

// Recalculate when width changes
watch([availableBars, bucketSize], () => {
  recalculate();
  changeCounter.value = -1;
});

function recalculate() {
  if (chartData.length <= availableBars.value || availableBars.value === 0) {
    downsampledData.value = [...chartData];
    return;
  }

  const size = bucketSize.value;
  const result = [];

  // Create complete buckets
  const numCompleteBuckets = Math.floor(chartData.length / size);

  for (let i = 0; i < numCompleteBuckets; i++) {
    const start = i * size;
    const end = start + size;
    const bucket = chartData.slice(start, end);
    const avg = bucket.reduce((sum, val) => sum + val, 0) / bucket.length;
    result.push(avg);
  }

  // Show only the last N bars that fit on screen
  downsampledData.value = result.slice(-availableBars.value);
}

function onBarHover(index: number) {
  // Map downsampled index back to original data index
  const numCompleteBuckets = Math.floor(chartData.length / bucketSize.value);
  const offset = Math.max(0, numCompleteBuckets - availableBars.value);
  const originalIndex = (offset + index) * bucketSize.value + bucketSize.value - 1;
  hoverIndex(Math.min(originalIndex, chartData.length - 1));
}
</script>
