<template>
  <div
    class="banner fixed top-0 right-0 left-0 z-(--vp-z-index-layout-top) flex items-center overflow-hidden bg-[oklch(74%_0.16_232.661)] p-4 font-bold text-gray-800 dark:bg-[oklch(60%_0.126_221.723)] dark:text-white"
  >
    <div class="mx-auto flex items-center gap-2 lg:gap-4">
      <span class="animate-bounce">🚀</span>K8s users, I need your help! I have added K8s support to Dozzle and I need
      your feedback! 🙏
      <a
        href="https://github.com/amir20/dozzle/discussions/3614"
        target="_blank"
        class="btn btn-sm btn-primary text-white!"
      >
        See the discussion
      </a>
    </div>
    <button class="btn btn-circle btn-ghost btn-xs ml-auto" @click="dismiss">
      <svg class="swap-on fill-current" xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 512 512">
        <polygon
          points="400 145.49 366.51 112 256 222.51 145.49 112 112 145.49 222.51 256 112 366.51 145.49 400 256 289.49 366.51 400 400 366.51 289.49 256 400 145.49"
        />
      </svg>
    </button>
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
