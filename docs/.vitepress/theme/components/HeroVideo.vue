<script lang="ts" setup>
import { useMutationObserver } from "@vueuse/core";
import { onMounted, onUnmounted, ref } from "vue";
import { data } from "../../starCounter.data.ts";

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

console.log(data.stars);
</script>

<template>
  <div
    class="border-rounded-md border-light-100 dark:border-dark-50 overflow-hidden border border-solid bg-[#eee] drop-shadow-md dark:bg-[#222]"
  >
    <video muted loop autoplay playsinline poster="../media/poster-dark.png" v-if="isDark">
      <source src="../media/dozzle-dark.mp4" type="video/mp4" />
      <img src="../media/poster-dark.png" alt="" />
    </video>
    <video muted loop autoplay playsinline poster="../media/poster-light.png" v-else>
      <source src="../media/dozzle-light.mp4" type="video/mp4" />
      <img src="../media/poster-light.png" alt="" />
    </video>
  </div>
</template>

<style scoped></style>
