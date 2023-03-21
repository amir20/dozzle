<script lang="ts" setup>
import { useMutationObserver } from "@vueuse/core";
import { onMounted, ref } from "vue";

const isDark = ref(false);
onMounted(() => {
  isDark.value = document.documentElement.classList.contains("dark");
});

useMutationObserver(
  document.documentElement,
  (mutations) => {
    isDark.value = document.documentElement.classList.contains("dark");
  },
  {
    attributes: true,
    attributeFilter: ["class"],
  }
);
</script>

<template>
  <browser-window drop-shadow-md>
    <video muted loop autoplay playsinline poster="../media/poster.png" v-if="isDark">
      <source src="../media/dozzle-dark.webm" type="video/webm" />
      <source src="../media/dozzle-dark.mp4" type="video/mp4" />
      <img src="../media/poster.png" alt="" />
    </video>
    <video muted loop autoplay playsinline poster="../media/poster.png" v-else>
      <source src="../media/dozzle-light.webm" type="video/webm" />
      <source src="../media/dozzle-light.mp4" type="video/mp4" />
      <img src="../media/poster.png" alt="" />
    </video>
  </browser-window>
</template>

<style scoped></style>
