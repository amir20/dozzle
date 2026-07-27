<script lang="ts" setup>
import { usePreferredReducedMotion } from "@vueuse/core";
import { useData } from "vitepress";
import { computed, onMounted, ref } from "vue";

import darkPoster from "../media/poster-dark.webp";
import lightPoster from "../media/poster-light.webp";
import darkVideo from "../media/dozzle-dark.mp4";
import lightVideo from "../media/dozzle-light.mp4";

const { isDark } = useData();
const reducedMotion = usePreferredReducedMotion();

// The posters are rendered on the server and toggled with CSS so the first paint
// is already the right theme. The video only mounts on the client, which keeps
// markup identical between SSR and hydration and avoids downloading both files.
const mounted = ref(false);
onMounted(() => (mounted.value = true));

const playVideo = computed(() => mounted.value && reducedMotion.value !== "reduce");
const video = computed(() => (isDark.value ? darkVideo : lightVideo));
const poster = computed(() => (isDark.value ? darkPoster : lightPoster));

const alt = "The Dozzle interface streaming container logs in real time";
</script>

<template>
  <div
    class="relative overflow-hidden rounded-md border border-solid border-gray-200 bg-[#eee] drop-shadow-md dark:border-gray-900 dark:bg-[#222]"
  >
    <!-- `!` is required: an unlayered `img { display: block }` reset outranks Tailwind's layered utilities. -->
    <img :src="lightPoster" width="1280" height="760" :alt="alt" class="block! w-full dark:hidden!" />
    <img :src="darkPoster" width="1280" height="760" :alt="alt" class="hidden! w-full dark:block!" />
    <video
      v-if="playVideo"
      :key="video"
      class="absolute inset-0 size-full"
      muted
      loop
      autoplay
      playsinline
      preload="metadata"
      aria-hidden="true"
      :poster="poster"
    >
      <source :src="video" type="video/mp4" />
    </video>
  </div>
</template>
