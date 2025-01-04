<template>
  <div class="hover:text-secondary relative" @mouseenter="mouseOver = true" @mouseleave="mouseOver = false">
    <div class="border-primary hidden overflow-hidden rounded-xs border px-px pt-1 pb-px md:flex">
      <StatSparkline :data="data" @selected-point="onSelectedPoint" />
    </div>
    <div class="bg-base inline-flex gap-1 rounded-sm p-px text-xs md:absolute md:-top-2 md:-left-0.5">
      <div class="font-light uppercase">{{ label }}</div>
      <div class="font-bold select-none">
        {{ mouseOver ? (selectedPoint?.value ?? selectedPoint?.y ?? statValue) : statValue }}
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
