<template>
  <div class="scroll-progress">
    <svg width="100" height="100" viewBox="0 0 100 100">
      <circle
        stroke="#00d1b2"
        stroke-width="3"
        fill="#000"
        fill-opacity="0.6"
        r="44"
        cx="50"
        cy="50"
        :style="{ '--progress': scrollProgress }"
      />
    </svg>
    <div class="percent columns is-vcentered is-centered">
      <span class="column is-narrow is-paddingless is-size-2">
        {{ Math.ceil(scrollProgress * 100) }}
      </span>
      <span class="column is-narrow is-paddingless">
        %
      </span>
    </div>
  </div>
</template>

<script>
import { mapState } from "vuex";
import throttle from "lodash.throttle";

export default {
  name: "ScrollProgress",
  data() {
    return {
      scrollProgress: 0,
      animation: { cancel: () => {} },
      parentElement: document,
    };
  },
  created() {
    this.onScrollThrottled = throttle(this.onScroll, 150);
  },
  mounted() {
    this.attachEvents();
    this.$once("hook:beforeDestroy", this.detachEvents);
  },
  watch: {
    activeContainers() {
      this.detachEvents();
      this.attachEvents();
    },
  },
  computed: {
    ...mapState(["activeContainers"]),
  },
  methods: {
    attachEvents() {
      this.parentElement = this.$el.closest("[data-scrolling]") || document;
      this.parentElement.addEventListener("scroll", this.onScrollThrottled);
    },
    detachEvents() {
      this.parentElement.removeEventListener("scroll", this.onScrollThrottled);
    },
    onScroll() {
      const p = this.parentElement == document ? document.documentElement : this.parentElement;
      this.scrollProgress = p.scrollTop / (p.scrollHeight - p.clientHeight);
      this.animation.cancel();
      this.animation = this.$el.animate(
        { opacity: [1, 0] },
        {
          duration: 500,
          delay: 2000,
          fill: "both",
          easing: "ease-out",
        }
      );
    },
  },
};
</script>
<style scoped lang="scss">
.scroll-progress {
  display: inline-block;
  position: relative;
  circle {
    transition: stroke-dashoffset 0.35s ease-out;
    transform: rotate(-90deg);
    transform-origin: 50% 50%;
    stroke-dashoffset: calc(276.32 - var(--progress) * 276.32);
    stroke-dasharray: 276.32 276.32;
    will-change: stroke-dashoffset;
  }

  .percent {
    position: absolute;
    left: 0;
    top: 0;
    right: 0;
    bottom: 0;
  }
}
</style>
