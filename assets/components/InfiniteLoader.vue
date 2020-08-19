<template>
  <div ref="observer" class="infinte-loader">
    <div class="spinner" v-show="isLoading">
      <div class="bounce1"></div>
      <div class="bounce2"></div>
      <div class="bounce3"></div>
    </div>
  </div>
</template>

<script>
export default {
  name: "InfiniteLoader",
  data() {
    return {
      isLoading: false,
    };
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
          this.isLoading = true;
          await this.onLoadMore();
          this.isLoading = false;
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
