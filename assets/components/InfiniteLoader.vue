<template>
  <div ref="root" class="flex min-h-[1px] justify-center">
    <span class="loading loading-bars loading-md mt-4 text-primary" v-show="isLoading"></span>
  </div>
</template>

<script lang="ts" setup>
const { onLoadMore = () => Promise.resolve(), enabled } = defineProps<{
  onLoadMore: () => Promise<void>;
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
