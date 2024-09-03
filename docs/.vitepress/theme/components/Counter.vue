<script setup lang="ts">
import { computed, onMounted, ref } from "vue";

const {
  start,
  end,
  duration,
  formatter = (value: number) => value.toLocaleString(),
} = defineProps<{
  start: number;
  end: number;
  duration: number;
  formatter?: (value: number) => string;
}>();

const text = ref(0);

onMounted(() => {
  let startTimestamp: number | undefined = undefined;

  const step = (timestamp: number) => {
    if (!startTimestamp) startTimestamp = timestamp;
    const progress = Math.min((timestamp - startTimestamp) / duration, 1);

    text.value = Math.floor(progress * (end - start) + start);
    if (progress < 1) {
      requestAnimationFrame(step);
    }
  };

  requestAnimationFrame(step);
});

const formmated = computed(() => formatter(text.value));
</script>

<template>
  {{ formmated }}
</template>
