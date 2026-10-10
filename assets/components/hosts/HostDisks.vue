<template>
  <Popover
    hover
    placement="bottom-end"
    panel-class="rounded-box border-base-content/10 bg-base-200 w-72 border p-3 text-xs shadow-lg"
  >
    <template #trigger>
      <!-- empty title: the host chip's own "Host" tooltip would pop up over the panel -->
      <!-- UsageMeter's track and thresholds, inline: the component is a labelled
           block sized for a panel row, too tall for a header fact. The fill stays
           neutral below 70% so the bar only takes color when disk needs a look. -->
      <button
        type="button"
        title=""
        class="hover:bg-base-content/10 -mx-1 flex items-center gap-1.5 rounded px-1 transition-colors"
      >
        {{ $t("label.disk") }}
        <span class="bg-base-content/10 h-1.5 w-10 overflow-hidden rounded-full">
          <span
            class="block h-full rounded-full transition-[width] duration-500"
            :class="fill(fullest.percent)"
            :style="{ width: width(fullest.percent) }"
          ></span>
        </span>
        <span class="text-base-content/80 font-mono">{{ fullest.percent }}%</span>
      </button>
    </template>

    <div class="flex items-baseline justify-between gap-2">
      <span class="text-base-content/60 font-semibold tracking-wide uppercase">{{ $t("label.disk") }}</span>
      <span class="font-mono text-sm font-semibold">{{ fullest.percent }}%</span>
    </div>

    <ul class="mt-3 space-y-2.5">
      <li v-for="drive in drives" :key="drive.name">
        <div class="flex items-center gap-2">
          <ph:hard-drive class="size-4 shrink-0 opacity-60" />
          <span class="min-w-0 flex-1 truncate">{{ drive.name }}</span>
          <span class="text-base-content/40 font-mono [word-spacing:-0.4ch]">
            {{ size(drive.used) }} / {{ size(drive.total) }}
          </span>
          <span class="w-10 text-right font-mono">{{ drive.percent }}%</span>
        </div>
        <div class="bg-base-content/10 mt-1 ml-6 h-1 overflow-hidden rounded-full">
          <div
            class="h-full rounded-full transition-[width] duration-500 motion-reduce:transition-none"
            :class="fill(drive.percent)"
            :style="{ width: width(drive.percent) }"
          ></div>
        </div>
      </li>
    </ul>
  </Popover>
</template>

<script lang="ts" setup>
export type Drive = { name: string; total: number; used: number; percent: number };

// Docker's own disk first, then the extra drives; the header and the trigger show
// the fullest, since that is the one that will run out.
const { drives } = defineProps<{ drives: Drive[] }>();

const fullest = computed(() => drives.reduce((a, b) => (b.percent > a.percent ? b : a)));

const size = (bytes: number) => formatBytes(bytes, { decimals: 1 });
const width = (percent: number) => `${Math.min(percent, 100)}%`;
const fill = (percent: number) => (percent > 90 ? "bg-error" : percent > 70 ? "bg-warning" : "bg-base-content/40");
</script>
