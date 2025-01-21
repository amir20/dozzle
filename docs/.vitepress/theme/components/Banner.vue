<template>
  <div
    class="banner fixed top-0 right-0 left-0 z-(--vp-z-index-layout-top) flex items-center overflow-hidden bg-[oklch(74%_0.16_232.661)] p-4 font-bold text-gray-800 dark:bg-[oklch(60%_0.126_221.723)] dark:text-white"
  >
    <div class="mx-auto flex items-center gap-2 lg:gap-4">
      <span class="animate-bounce">👋🏼</span> Using Dozzle? I'd love to hear about your experience in this short survey
      to shape the future of Dozzle.
      <a href="https://tally.so/r/wLv4g2" target="_blank" class="btn btn-sm btn-primary btn-ghost text-white!">
        Take a quick survey 🙏🏼
      </a>
    </div>
    <button class="btn btn-circle btn-xs ml-auto" @click="dismiss">x</button>
  </div>
</template>
<script setup>
import { onMounted } from "vue";
const dismiss = () => {
  localStorage.setItem("banner-dismissed", (Date.now() + 8.64e7 * 1).toString());
  document.documentElement.classList.add("banner-dismissed");
};

onMounted(() => {
  if (new Date(+localStorage.getItem("banner-dismissed")) > Date.now()) {
    document.documentElement.classList.add("banner-dismissed");
  }
});
</script>
<style>
.banner-dismissed {
  --vp-layout-top-height: 0px !important;
}
@media (min-width: 375px) {
  html {
    --vp-layout-top-height: 100px;
  }
}
@media (min-width: 768px) {
  html {
    --vp-layout-top-height: 63px;
  }
}
</style>
<style scoped>
.banner-dismissed .banner {
  display: none;
}
</style>
