<template>
  <div class="banner fixed top-0 right-0 left-0 z-(--vp-z-index-layout-top) flex overflow-hidden bg-[#383636] p-2">
    Thiis is a test
    <button class="btn btn-circle btn-xs ml-auto" @click="dismiss">
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
        <path
          d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z"
        />
      </svg>
    </button>
  </div>
</template>
<script setup>
const dismiss = () => {
  localStorage.setItem("banner-dismissed", (Date.now() + 8.64e7 * 1).toString());
  document.documentElement.classList.add("banner-dismissed");
};
</script>
<style>
.banner-dismissed {
  --vp-layout-top-height: 0px !important;
}
html {
  --vp-layout-top-height: 88px;
}
@media (min-width: 375px) {
  html {
    --vp-layout-top-height: 64px;
  }
}
@media (min-width: 768px) {
  html {
    --vp-layout-top-height: 40px;
  }
}
</style>
<style scoped>
.banner-dismissed .banner {
  display: none;
}
</style>
