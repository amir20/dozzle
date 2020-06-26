<template>
  <section :class="{ 'is-full-height-scrollable': scrollable }">
    <header v-if="$slots.header">
      <slot name="header"></slot>
    </header>
    <main ref="content" :data-scrolling="scrollable">
      <div class="scrollbar-progress is-hidden-mobile">
        <scroll-progress v-show="paused"></scroll-progress>
      </div>
      <slot></slot>
      <div ref="scrollObserver"></div>
    </main>

    <div class="scrollbar-notification">
      <transition name="fade">
        <button class="button" :class="hasMore ? 'has-more' : ''" @click="scrollToBottom('instant')" v-show="paused">
          <icon name="download"></icon>
        </button>
      </transition>
    </div>
  </section>
</template>

<script>
import Icon from "./Icon";
import ScrollProgress from "./ScrollProgress";

export default {
  props: {
    scrollable: {
      type: Boolean,
      default: true,
    },
  },
  components: {
    Icon,
    ScrollProgress,
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
    const mutationObserver = new MutationObserver((e) => {
      if (!this.paused) {
        this.scrollToBottom("instant");
      } else {
        this.hasMore = true;
      }
    });
    mutationObserver.observe(content, { childList: true, subtree: true });
    this.$once("hook:beforeDestroy", () => mutationObserver.disconnect());

    const intersectionObserver = new IntersectionObserver(
      (entries) => (this.paused = entries[0].intersectionRatio == 0),
      { threshholds: [0, 1], rootMargin: "80px 0px" }
    );
    intersectionObserver.observe(this.$refs.scrollObserver);
    this.$once("hook:beforeDestroy", () => intersectionObserver.disconnect());
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

  .scrollbar-progress {
    text-align: right;
    margin-right: 110px;
    .scroll-progress {
      position: fixed;
      top: 60px;
      z-index: 2;
    }
  }

  .scrollbar-notification {
    text-align: right;
    margin-right: 65px;
    button {
      position: fixed;
      bottom: 30px;
      background-color: var(--secondary-color);
      transition: background-color 1s ease-out;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12), 0 1px 2px rgba(0, 0, 0, 0.24);
      border: none !important;

      &.has-more {
        background-color: var(--primary-color);
        animation-name: bounce;
        animation-duration: 1000ms;
        animation-fill-mode: both;
        color: #fff;
      }
    }
  }

  @keyframes bounce {
    0%,
    20%,
    50%,
    80%,
    100% {
      transform: translateY(0);
    }
    40% {
      transform: translateY(-30px);
    }
    60% {
      transform: translateY(-15px);
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
