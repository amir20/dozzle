<template lang="html">
  <div ref="observer"></div>
</template>

<script>
export default {
  name: "InfiniteLoader",
  data() {
    return {
      scrollingParent: null
    };
  },
  props: {
    onLoadMore: Function,
    enabled: Boolean
  },
  mounted() {
    this.scrollingParent = this.$el.closest("[data-scrolling]");
    const intersectionObserver = new IntersectionObserver(
      async entries => {
        if (entries[0].intersectionRatio <= 0) return;
        if (this.onLoadMore && this.enabled) {
          const previousHeight = this.scrollingParent.scrollHeight;
          await this.onLoadMore();
          this.$nextTick(() => (this.scrollingParent.scrollTop += this.scrollingParent.scrollHeight - previousHeight));
        }
      },
      { threshholds: 1 }
    );

    intersectionObserver.observe(this.$refs.observer);
  }
};
</script>
<style scoped lang="scss"></style>
