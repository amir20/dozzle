<template>
  <div ref="chartContainer" class="flex items-end gap-[2px]" @mousemove="onContainerHover">
    <div
      v-for="(dataPoint, i) in downsampledData"
      :key="i"
      class="min-h-px flex-1 rounded-t-sm"
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

const hoverIndex = defineEmit<[startIndex: number, endIndex: number]>();

const chartContainer = ref<HTMLElement | null>(null);
const { width: containerWidth } = useElementSize(chartContainer, undefined, { box: "content-box" });
const isVisible = useElementVisibility(chartContainer);

const BAR_WIDTH = 3;
const GAP = 2;

const availableBars = computed(() => Math.floor(containerWidth.value / (BAR_WIDTH + GAP)));
const bucketSize = computed(() => Math.ceil(chartData.length / availableBars.value));

const downsampledData = ref<number[]>([]);
const changeCounter = ref(0);

// Watch chartData reference changes (new array from .map())
watch(
  () => chartData,
  () => {
    if (!isVisible.value) return;

    changeCounter.value++;
    if (changeCounter.value >= bucketSize.value) {
      recalculate();
      changeCounter.value = 0;
    }
  },
  { immediate: true },
);

// Recalculate when becoming visible
watch(isVisible, (visible) => {
  if (visible) {
    recalculate();
    changeCounter.value = 0;
  }
});

// Recalculate when width changes significantly
watch([availableBars, bucketSize], () => {
  if (isVisible.value) {
    recalculate();
  }
});

function recalculate() {
  if (chartData.length <= availableBars.value || availableBars.value === 0) {
    downsampledData.value = [...chartData];
    return;
  }

  const size = bucketSize.value;
  const result: number[] = [];

  // Create complete buckets
  const numCompleteBuckets = Math.floor(chartData.length / size);

  for (let i = 0; i < numCompleteBuckets; i++) {
    const start = i * size;
    const end = start + size;
    let sum = 0;
    for (let j = start; j < end; j++) {
      sum += chartData[j];
    }
    result.push(sum / size);
  }

  // Show only the last N bars that fit on screen
  downsampledData.value = result.slice(-availableBars.value);
}

function onContainerHover(event: MouseEvent) {
  if (!chartContainer.value) return;

  const rect = chartContainer.value.getBoundingClientRect();
  const x = event.clientX - rect.left;

  // Calculate which bar the mouse is over based on position
  const barWidth = containerWidth.value / downsampledData.value.length;
  const index = Math.floor(x / barWidth);

  // Ensure index is within bounds
  if (index < 0 || index >= downsampledData.value.length) return;

  // Map downsampled index back to original data index range
  const numCompleteBuckets = Math.floor(chartData.length / bucketSize.value);
  const offset = Math.max(0, numCompleteBuckets - availableBars.value);
  const startIndex = (offset + index) * bucketSize.value;
  const endIndex = Math.min(startIndex + bucketSize.value - 1, chartData.length - 1);
  hoverIndex(startIndex, endIndex);
}
</script>
