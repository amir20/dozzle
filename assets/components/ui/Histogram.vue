<template>
  <div class="flex flex-col gap-1">
    <!-- pt-4 is headroom for the count above the tallest bar. -->
    <div class="flex h-28 items-end gap-0.5 pt-4" @mouseleave="leave">
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
          class="relative w-full rounded-t-sm transition-colors"
          :class="
            bin.count === 0
              ? 'bg-base-content/10'
              : hovered === i
                ? 'bg-primary'
                : 'bg-primary/70 group-hover:bg-primary'
          "
          :style="{ height: heightOf(bin.count) }"
        >
          <span
            v-if="bin.count > 0 && (labelled || hovered === i)"
            class="text-base-content/60 absolute inset-x-0 bottom-full mb-0.5 truncate text-center font-mono text-[0.65rem] tabular-nums"
            data-testid="histogram-count"
          >
            {{ bin.count.toLocaleString() }}
          </span>
        </div>
      </div>
    </div>
    <!-- Few enough bins to name each one's range; otherwise only the two ends. -->
    <div v-if="ranged" class="text-base-content/40 flex gap-0.5 font-mono text-xs tabular-nums">
      <span v-for="(bin, i) in bins" :key="i" class="min-w-0 flex-1 truncate text-center" :title="range(bin)">
        {{ range(bin) }}
      </span>
    </div>
    <div class="text-base-content/40 flex items-baseline gap-2 font-mono text-xs tabular-nums">
      <span v-if="!ranged">{{ format(bins[0].start) }}</span>
      <span v-if="label" class="text-base-content/50 mx-auto truncate">{{ label }} &rarr;</span>
      <span v-if="!ranged" :class="{ 'ml-auto': !label }">{{ format(bins[bins.length - 1].end) }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { HistogramBin } from "./histogramBins";

const {
  bins,
  format = String,
  label,
} = defineProps<{
  bins: HistogramBin[];
  format?: (value: number) => string;
  // What the x-axis measures, e.g. the column name.
  label?: string;
}>();

const emit = defineEmits<{ hoverBin: [bin: HistogramBin]; hoverEnd: [] }>();
const hoverBin = (bin: HistogramBin) => emit("hoverBin", bin);
const hoverEnd = () => emit("hoverEnd");

const hovered = ref<number | null>(null);
const peak = computed(() => Math.max(1, ...bins.map((b) => b.count)));
// Past these the numbers crowd into each other, so they show on hover only.
const labelled = computed(() => bins.length <= 16);
const ranged = computed(() => bins.length <= 8);

// Scaled to the tallest bin with no headroom, and never thinner than a sliver:
// a single outlier next to a bin of hundreds is the reason to look at a
// histogram, and at true scale it would round to nothing.
function heightOf(count: number) {
  if (count === 0) return "2px";
  return `max(3px, ${(count / peak.value) * 100}%)`;
}

function range(bin: HistogramBin) {
  return bin.start === bin.end ? format(bin.start) : `${format(bin.start)}–${format(bin.end)}`;
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
