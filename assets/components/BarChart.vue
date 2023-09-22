<template>
  <div class="relative">
    <div class="bar h-7 origin-left rounded-br rounded-tr bg-primary transition-transform"></div>
    <div class="absolute inset-0 flex flex-col justify-center px-2 text-sm">
      <slot></slot>
    </div>
  </div>
</template>

<script lang="ts" setup>
const { value } = defineProps<{ value: number }>();

const minValue = computed(() => Math.min(value, 1));
</script>

<style scoped>
.bar {
  transform: scaleX(v-bind(minValue));
}
</style>
