<template lang="html">
  <div ref="observer" class="control" :class="{ 'is-loading': isLoading }"></div>
</template>

<script>
export default {
  name: "InfiniteLoader",
  data() {
    return {
      scrollingParent: null,
      isLoading: false
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
          this.isLoading = true;
          await this.onLoadMore();
          this.isLoading = false;
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
