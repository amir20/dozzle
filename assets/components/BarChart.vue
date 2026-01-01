<template>
  <div ref="chartContainer" class="flex items-end gap-[2px]">
    <div
      v-for="(dataPoint, i) in downsampledData"
      :key="i"
      class="flex-1 rounded-t-sm"
      :class="barClass"
      :style="`height: ${Math.min(dataPoint, 100)}%`"
    ></div>
  </div>
</template>

<script setup lang="ts">
const { chartData, barClass = "" } = defineProps<{
  chartData: number[];
  barClass?: string;
}>();

const chartContainer = ref<HTMLElement | null>(null);
const { width } = useElementSize(chartContainer);

const downsampledData = computed(() => {
  const BAR_WIDTH = 3;
  const GAP = 2;
  const availableBars = Math.floor(width.value / (BAR_WIDTH + GAP));

  if (chartData.length <= availableBars || availableBars === 0) {
    return chartData;
  }

  // Downsample by averaging buckets
  const bucketSize = chartData.length / availableBars;
  const result = [];
  for (let i = 0; i < availableBars; i++) {
    const start = Math.floor(i * bucketSize);
    const end = Math.floor((i + 1) * bucketSize);
    const bucket = chartData.slice(start, end);
    const avg = bucket.reduce((sum, val) => sum + val, 0) / bucket.length;
    result.push(avg);
  }
  return result;
});
</script>
