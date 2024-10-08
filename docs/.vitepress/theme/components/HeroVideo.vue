<script lang="ts" setup>
import { useMutationObserver } from "@vueuse/core";
import { onMounted, ref } from "vue";

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
</script>

<template>
  <div
    class="overflow-hidden rounded-md border border-solid border-gray-200 bg-[#eee] text-[red] drop-shadow-md dark:border-gray-900 dark:bg-[#222]"
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
