<template lang="html">
  <section :class="{ 'is-full-height-scrollable': scrollable }">
    <header v-if="$slots.header">
      <slot name="header"></slot>
    </header>
    <main ref="content" :data-scrolling="scrollable">
      <slot></slot>
      <div ref="scrollObserver"></div>
    </main>
    <div class="scroll-bar-notification">
      <transition name="fade">
        <button
          class="button"
          :class="hasMore ? 'is-warning' : 'is-primary'"
          @click="scrollToBottom('instant')"
          v-show="paused"
        >
          <icon name="download"></icon>
        </button>
      </transition>
    </div>
  </section>
</template>

<script>
import Icon from "./Icon";

export default {
  props: {
    scrollable: {
      type: Boolean,
      default: true,
    },
  },
  components: {
    Icon,
  },
  name: "ScrollableView",
  data() {
    return {
      paused: false,
      hasMore: false,
    };
  },
  mounted() {
    const { content } = this.$refs;
    new MutationObserver((e) => {
      if (!this.paused) {
        // this.scrollToBottom("instant");
      } else {
        this.hasMore = true;
      }
    }).observe(content, { childList: true, subtree: true });

    const intersectionObserver = new IntersectionObserver(
      (entries) => (this.paused = entries[0].intersectionRatio == 0),
      { threshholds: [0, 1], rootMargin: "80px 0px" }
    );

    intersectionObserver.observe(this.$refs.scrollObserver);
  },

  methods: {
    scrollToBottom(behavior = "instant") {
      this.$refs.scrollObserver.scrollIntoView({ behavior });
      this.hasMore = false;
    },
  },
};
</script>
<style scoped lang="scss">
section {
  display: flex;
  flex-direction: column;

  &.is-full-height-scrollable {
    height: 100vh;
  }

  main {
    flex: 1;
    overflow: auto;
    scroll-snap-type: y proximity;
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
