<script lang="ts" setup>
import { useMutationObserver } from "@vueuse/core";
import { onMounted, onUnmounted, ref } from "vue";

const isDark = ref(false);
onMounted(() => {
  isDark.value = document.documentElement.classList.contains("dark");
  useMutationObserver(
    document.documentElement,
    (mutations) => {
      isDark.value = document.documentElement.classList.contains("dark");
    },
    {
      attributes: true,
      attributeFilter: ["class"],
    },
  );
});

onMounted(() => {
  document.documentElement.classList.add("home");
});

onUnmounted(() => {
  document.documentElement.classList.remove("home");
});
</script>

<template>
  <video muted loop autoplay playsinline poster="../media/poster.png" v-if="isDark" class="drop-shadow-md">
    <source src="../media/dozzle-dark.mp4" type="video/mp4" />
    <img src="../media/poster.png" alt="" />
  </video>
</template>

<style scoped></style>
