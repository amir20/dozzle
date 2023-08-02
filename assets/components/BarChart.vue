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
  height: 1.5em;
  background-color: var(--primary-color);
  transform-origin: left;
  transform: scaleX(v-bind(minValue));
  transition: transform 0.2s ease-out;
  border-top-right-radius: 0.2em;
  border-bottom-right-radius: 0.2em;
}

.is-overlay {
  font-size: 0.9em;
  padding: 0 0.5em;
}
</style>
