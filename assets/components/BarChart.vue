<template>
  <div>
    <div class="bar"></div>
  </div>
</template>

<script lang="ts" setup>
const { value } = defineProps<{ value: number }>();
</script>

<style scoped>
.bar {
  height: 100%;
  background-color: var(--primary-color);
  transform-origin: left;
  transform: scaleX(v-bind(value));
  transition: transform 0.2s ease-out;
}
</style>
