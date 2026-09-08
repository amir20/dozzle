<template>
  <div class="flex min-w-0 flex-col justify-center gap-1 px-3 py-1.5" :title="title" @mouseleave="hoveredValue = null">
    <div class="flex items-baseline gap-1.5 leading-none">
      <span class="text-base-content/40 shrink-0 text-[10px] font-medium tracking-wider uppercase">{{ label }}</span>
      <slot name="value" :hoveredValue="hoveredValue" />
    </div>
    <slot name="chart" :onHoverValue="(v: number) => (hoveredValue = v)" />
  </div>
</template>

<script lang="ts" setup>
defineProps<{
  label: string;
  title?: string;
}>();

const hoveredValue = ref<number | null>(null);
</script>
