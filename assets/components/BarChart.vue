<template>
  <div class="is-relative">
    <div class="bar"></div>
    <div class="is-overlay">
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
  height: 100%;
  background-color: var(--primary-color);
  transform-origin: left;
  transform: scaleX(v-bind(minValue));
  transition: transform 0.2s ease-out;
}
</style>
