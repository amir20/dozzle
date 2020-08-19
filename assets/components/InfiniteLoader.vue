<template>
  <div ref="observer" class="control"></div>
</template>

<script>
export default {
  name: "InfiniteLoader",
  data() {
    return {};
  },
  props: {
    onLoadMore: Function,
    enabled: Boolean,
  },
  mounted() {
    const intersectionObserver = new IntersectionObserver(
      async (entries) => {
        if (entries[0].intersectionRatio <= 0) return;
        if (this.onLoadMore && this.enabled) {
          const scrollingParent = this.$el.closest("[data-scrolling]") || document.documentElement;
          const previousHeight = scrollingParent.scrollHeight;
          await this.onLoadMore();
          this.$nextTick(() => (scrollingParent.scrollTop += scrollingParent.scrollHeight - previousHeight));
        }
      },
      { threshholds: 1 }
    );

    intersectionObserver.observe(this.$refs.observer);

    this.$once("hook:beforeDestroy", () => intersectionObserver.disconnect());
  },
};
</script>
<style scoped lang="scss"></style>
