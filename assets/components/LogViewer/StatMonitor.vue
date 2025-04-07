<template>
  <div class="hover:text-secondary relative" @mouseenter="mouseOver = true" @mouseleave="mouseOver = false">
    <div class="border-primary overflow-hidden rounded-xs border px-px pt-1 pb-px max-md:hidden">
      <StatSparkline :data="data" @selected-point="onSelectedPoint" />
    </div>
    <div class="bg-base-200 inline-flex gap-1 rounded-sm p-px text-xs md:absolute md:-top-2 md:-left-0.5">
      <div class="font-light uppercase">{{ label }}</div>
      <div class="font-bold select-none">
        {{ mouseOver ? (selectedPoint?.value ?? selectedPoint?.y ?? statValue) : statValue }}
        <span v-if="limit !== -1 && !mouseOver" class="max-md:hidden"> / {{ limit }} </span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
const {
  data,
  label,
  statValue,
  limit = -1,
} = defineProps<{
  data: Point<unknown>[];
  label: string;
  statValue: string | number;
  limit?: string | number;
}>();
const selectedPoint = ref<Point<unknown> | undefined>();
const onSelectedPoint = (point: Point<unknown>) => (selectedPoint.value = point);
const mouseOver = ref(false);
</script>
