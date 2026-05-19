<template>
  <div
    class="flex min-w-0 flex-col gap-px rounded-md px-2.5 py-1"
    :class="cardClass"
    :title="title"
    @mouseleave="hoveredValue = null"
  >
    <div class="flex h-[18px] items-center gap-2 text-[12.5px] font-medium">
      <component :is="icon" class="size-3.5 shrink-0" :class="iconClass" />
      <slot name="value" :hoveredValue="hoveredValue" />
    </div>
    <slot name="chart" :onHoverValue="(v: number) => (hoveredValue = v)" />
  </div>
</template>

<script lang="ts" setup>
import type { Component } from "vue";

defineProps<{
  icon: Component;
  cardClass?: string;
  iconClass?: string;
  title?: string;
}>();

const hoveredValue = ref<number | null>(null);
</script>
