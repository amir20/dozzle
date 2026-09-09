<template>
  <!--
    The one meter every cloud surface draws: the popover, the settings card and the
    destination drawer. A daisyUI <progress> is taller than the type it sits under
    and carries its own theme colors, so the bar is a plain div pair scaled to the
    row instead.
  -->
  <div class="flex flex-col" :class="compact ? 'gap-1.5' : 'gap-2'">
    <div class="flex items-baseline justify-between gap-2">
      <span class="text-base-content/60" :class="compact ? 'text-xs' : 'text-sm'">{{ $t("cloud.usage") }}</span>
      <span class="font-mono" :class="compact ? 'text-xs' : 'text-sm'">
        <span class="font-semibold">{{ used.toLocaleString() }}</span>
        <span class="text-base-content/40"> / {{ limit.toLocaleString() }}</span>
      </span>
    </div>
    <div class="bg-base-content/10 h-1.5 w-full overflow-hidden rounded-full">
      <div
        class="h-full rounded-full transition-[width] duration-500"
        :class="percent > 90 ? 'bg-error' : percent > 70 ? 'bg-warning' : 'bg-primary'"
        :style="{ width: `${Math.min(percent, 100)}%` }"
      ></div>
    </div>
    <div
      class="text-base-content/40 flex justify-between gap-2 font-mono"
      :class="compact ? 'text-[0.6875rem]' : 'text-xs'"
    >
      <span class="truncate">{{ period }}</span>
      <span class="shrink-0">{{ percent.toFixed(1) }}%</span>
    </div>
  </div>
</template>

<script lang="ts" setup>
const { used, limit } = defineProps<{
  used: number;
  limit: number;
  period?: string;
  /** Tightens type and spacing for the popover, which is a third of the width of the other two. */
  compact?: boolean;
}>();

// A plan without a limit would otherwise divide by zero and render a NaN-wide bar.
const percent = computed(() => (limit ? (used / limit) * 100 : 0));
</script>
