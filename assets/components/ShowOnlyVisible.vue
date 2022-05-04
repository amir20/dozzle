<template>
  <div ref="root">
    <slot v-if="isVisible"></slot>
  </div>
</template>

<script setup lang="ts">
import { useIntersectionObserver } from "@vueuse/core";
import { ref } from "vue";

const isVisible = ref(false);
const root = ref();
useIntersectionObserver(root, ([{ isIntersecting }], observerElement) => {
  if (isIntersecting && isVisible.value === false) {
    console.log("Intersecting");
    isVisible.value = isIntersecting;
  }
});
</script>
