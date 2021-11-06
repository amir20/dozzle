<template>
  <section :class="{ 'is-full-height-scrollable': scrollable }">
    <header v-if="$slots.header">
      <slot name="header"></slot>
    </header>
    <main ref="content" :data-scrolling="scrollable">
      <div class="is-scrollbar-progress is-hidden-mobile">
        <scroll-progress v-show="paused" :indeterminate="loading" :auto-hide="!loading"></scroll-progress>
      </div>
      <slot :setLoading="setLoading"></slot>
      <div ref="scrollObserver" class="is-scroll-observer"></div>
    </main>

    <div class="is-scrollbar-notification">
      <transition name="fade">
        <button
          class="button pl-1 pr-1"
          :class="hasMore ? 'has-more' : ''"
          @click="scrollToBottom('instant')"
          v-show="paused"
        >
          <chevron-double-down-icon />
        </button>
      </transition>
    </div>
  </section>
</template>

<script>
import ScrollProgress from "./ScrollProgress";
import ChevronDoubleDownIcon from "~icons/mdi-light/chevron-double-down";

export default {
  props: {
    scrollable: {
      type: Boolean,
      default: true,
    },
  },
  components: {
    ScrollProgress,
    ChevronDoubleDownIcon,
  },
  name: "ScrollableView",
  data() {
    return {
      paused: false,
      hasMore: false,
      loading: false,
      mutationObserver: null,
      intersectionObserver: null,
    };
  },
  mounted() {
    const { content } = this.$refs;
    this.mutationObserver = new MutationObserver((e) => {
      if (!this.paused) {
        this.scrollToBottom("instant");
      } else {
        const record = e[e.length - 1];
        if (
          record.target.children[record.target.children.length - 1] == record.addedNodes[record.addedNodes.length - 1]
        ) {
          this.hasMore = true;
        }
      }
    });
    this.mutationObserver.observe(content, { childList: true, subtree: true });

    this.intersectionObserver = new IntersectionObserver(
      (entries) => (this.paused = entries[0].intersectionRatio == 0),
      { threshholds: [0, 1], rootMargin: "80px 0px" }
    );
    this.intersectionObserver.observe(this.$refs.scrollObserver);
  },
  beforeUnmount() {
    this.mutationObserver.disconnect();
    this.intersectionObserver.disconnect();
  },
  methods: {
    scrollToBottom(behavior = "instant") {
      this.$refs.scrollObserver.scrollIntoView({ behavior });
      this.hasMore = false;
    },
    setLoading(loading) {
      this.loading = loading;
    },
  },
};
</script>
<style scoped lang="scss">
section {
  display: flex;
  flex-direction: column;

  header {
    position: sticky;
    top: 0;
    background: var(--body-background-color);
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  }

  &.is-full-height-scrollable {
    height: 100vh;
    min-height: 0;
  }

  main {
    flex: 1;
    overflow: auto;
    scroll-snap-type: y proximity;
  }

  .is-scrollbar-progress {
    text-align: right;
    margin-right: 110px;
    .scroll-progress {
      position: fixed;
      top: 60px;
      z-index: 2;
    }
  }

  .is-scroll-observer {
    height: 1px;
  }

  .is-scrollbar-notification {
    text-align: right;
    margin-right: 65px;
    button {
      position: fixed;
      bottom: 30px;
      background-color: var(--secondary-color);
      transition: background-color 1s ease-out;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12), 0 1px 2px rgba(0, 0, 0, 0.24);
      border: none !important;
      color: #222;

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
