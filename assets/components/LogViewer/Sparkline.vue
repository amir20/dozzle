<template>
  <div class="block h-5 w-full">
    <BarChart
      :chart-data="data"
      :bar-class="`${barClass} opacity-80 hover:opacity-100`"
      class="h-5 w-full"
      @hover-value="(value: number) => emit('hoverValue', value)"
    />
  </div>
</template>

<script lang="ts" setup>
import BarChart, { type BarDataPoint } from "@/components/BarChart.vue";

defineProps<{
  data: BarDataPoint[];
  barClass?: string;
}>();
const emit = defineEmits<{ hoverValue: [value: number] }>();
</script>
