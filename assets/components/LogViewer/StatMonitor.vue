<template>
  <div class="relative hover:text-secondary" @mouseenter="mouseOver = true" @mouseleave="mouseOver = false">
    <div class="hidden overflow-hidden rounded-sm border border-primary px-px pb-px pt-1 md:flex">
      <StatSparkline :data="data" @selected-point="onSelectedPoint" />
    </div>
    <div class="inline-flex gap-1 rounded bg-base p-px text-xs md:absolute md:-left-0.5 md:-top-2">
      <div class="font-light uppercase">{{ label }}</div>
      <div class="select-none font-bold">
        {{ mouseOver ? selectedPoint?.value ?? selectedPoint?.y ?? statValue : statValue }}
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
const { data, label, statValue } = defineProps<{ data: Point<unknown>[]; label: string; statValue: string | number }>();

let selectedPoint: Point<unknown> | undefined = $ref();

function onSelectedPoint(point: Point<unknown>) {
  selectedPoint = point;
}

let mouseOver = $ref(false);
</script>
