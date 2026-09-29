<template>
  <div class="flex flex-col gap-1">
    <div class="flex h-24 items-end gap-0.5" @mouseleave="leave">
      <!-- A column is the full height so the whole strip is the hover target, not
           just the painted bar: a one-row bin in the tail is a few pixels tall. -->
      <div
        v-for="(bin, i) in bins"
        :key="i"
        class="group flex h-full min-w-0 flex-1 items-end"
        data-testid="histogram-bin"
        @mouseenter="enter(i)"
        @touchstart.passive="enter(i)"
      >
        <div
          class="w-full rounded-t-sm transition-colors"
          :class="
            bin.count === 0
              ? 'bg-base-content/10'
              : hovered === i
                ? 'bg-primary'
                : 'bg-primary/70 group-hover:bg-primary'
          "
          :style="{ height: heightOf(bin.count) }"
        ></div>
      </div>
    </div>
    <div class="text-base-content/40 flex justify-between font-mono text-xs tabular-nums">
      <span>{{ format(bins[0].start) }}</span>
      <span>{{ format(bins[bins.length - 1].end) }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { HistogramBin } from "./histogramBins";

const { bins, format = String } = defineProps<{
  bins: HistogramBin[];
  format?: (value: number) => string;
}>();

const hoverBin = defineEmit<[bin: HistogramBin]>();
const hoverEnd = defineEmit<[]>();

const hovered = ref<number | null>(null);
const peak = computed(() => Math.max(1, ...bins.map((b) => b.count)));

// Scaled to the tallest bin with no headroom, and never thinner than a sliver:
// a single outlier next to a bin of hundreds is the reason to look at a
// histogram, and at true scale it would round to nothing.
function heightOf(count: number) {
  if (count === 0) return "2px";
  return `max(3px, ${(count / peak.value) * 100}%)`;
}

function enter(index: number) {
  hovered.value = index;
  hoverBin(bins[index]);
}

function leave() {
  hovered.value = null;
  hoverEnd();
}

watch(
  () => bins,
  () => (hovered.value = null),
);
</script>
