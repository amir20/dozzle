<template>
  <div class="scroll-progress">
    <svg width="120" height="120">
      <circle
        stroke="white"
        stroke-width="4"
        fill="#000"
        fill-opacity="0.6"
        r="52"
        cx="60"
        cy="60"
        :style="{ '--progress': scrollProgress }"
      />
    </svg>
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
      // this.animation.cancel();
      // this.animation = this.$refs.progress.animate(
      //   { opacity: [1, 0] },
      //   {
      //     duration: 500,
      //     delay: 2000,
      //     fill: "both",
      //     easing: "ease-out",
      //   }
      // );
    },
  },
};
</script>
<style scoped lang="scss">
.scroll-progress {
  position: fixed;

  circle {
    transition: stroke-dashoffset 0.35s ease-out;
    transform: rotate(-90deg);
    transform-origin: 50% 50%;
    stroke-dashoffset: calc(326.7256 - var(--progress) * 326.7256);
    stroke-dasharray: 326.7256 326.7256;
    will-change: stroke-dashoffset;
  }
}
</style>
