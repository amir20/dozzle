<template lang="html">
  <section>
    <header v-if="$slots.header">
      <slot name="header"></slot>
    </header>
    <main ref="content" @scroll.passive="onScroll">
      <slot></slot>
    </main>
    <div class="scroll-bar-notification">
      <transition name="fade">
        <button
          class="button"
          :class="hasMore ? 'is-warning' : 'is-primary'"
          @click="scrollToBottom('instant')"
          v-show="paused"
        >
          <span class="icon large"> <i class="fas fa-chevron-down"></i> </span>
        </button>
      </transition>
    </div>
  </section>
</template>

<script>
export default {
  name: "ScrollableView",
  data() {
    return {
      paused: false,
      hasMore: false
    };
  },
  mounted() {
    const { content } = this.$refs;
    new MutationObserver(e => {
      if (!this.paused) {
        this.scrollToBottom("smooth");
      } else {
        this.hasMore = true;
      }
    }).observe(content, { childList: true, subtree: true });
  },
  methods: {
    scrollToBottom(behavior = "instant") {
      const { content } = this.$refs;
      if (typeof content.scroll === "function") {
        content.scroll({ top: content.scrollHeight, behavior });
      } else {
        content.scrollTop = content.scrollHeight;
      }
      this.hasMore = false;
    },
    onScroll(e) {
      const { content } = this.$refs;
      this.paused = content.scrollTop + content.clientHeight + 1 < content.scrollHeight;
    }
  },
  watch: {}
};
</script>
<style scoped lang="scss">
section {
  display: flex;
  flex-direction: column;
  height: 100vh;

  main {
    flex: 1;
    overflow: auto;
  }
  .scroll-bar-notification {
    text-align: right;
    margin-right: 65px;
    button {
      position: fixed;
      bottom: 30px;
    }
  }

  .fade-enter-active,
  .fade-leave-active {
    transition: opacity 0.15s ease-in;
  }
  .fade-enter,
  .fade-leave-to {
    opacity: 0;
  }
}
</style>
