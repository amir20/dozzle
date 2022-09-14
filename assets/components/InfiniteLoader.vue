<template>
  <div ref="root" class="infinte-loader">
    <div class="spinner" v-show="isLoading">
      <div class="bounce1"></div>
      <div class="bounce2"></div>
      <div class="bounce3"></div>
    </div>
  </div>
</template>

<script lang="ts" setup>
const { onLoadMore = () => {}, enabled } = defineProps<{
  onLoadMore: () => void;
  enabled: boolean;
}>();

const isLoading = ref(false);
const root = ref<HTMLElement>();

const observer = new IntersectionObserver(async (entries) => {
  if (entries[0].intersectionRatio <= 0) return;
  if (onLoadMore && enabled) {
    const scrollingParent = root.value?.closest("[data-scrolling]") || document.documentElement;
    const previousHeight = scrollingParent.scrollHeight;
    isLoading.value = true;
    await onLoadMore();
    isLoading.value = false;
    await nextTick();
    scrollingParent.scrollTop += scrollingParent.scrollHeight - previousHeight;
  }
});

onMounted(() => observer.observe(root.value!));
onUnmounted(() => observer.disconnect());
</script>

<style scoped lang="scss">
.infinte-loader {
  min-height: 1px;
}
.spinner {
  margin: 10px auto 0;
  width: 70px;
  text-align: center;

  & > div {
    width: 12px;
    height: 12px;
    background-color: var(--primary-color);
    border-radius: 100%;
    display: inline-block;
    animation: sk-bouncedelay 0.8s infinite ease-in-out both;
  }
  & .bounce1 {
    animation-delay: -0.32s;
  }

  & .bounce2 {
    animation-delay: -0.16s;
  }
}

@keyframes sk-bouncedelay {
  0%,
  80%,
  100% {
    transform: scale(0);
  }
  40% {
    transform: scale(1);
  }
}
</style>
