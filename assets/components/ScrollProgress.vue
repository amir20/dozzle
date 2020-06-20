<template>
  <div class="progress" :style="{ '--progress': scrollProgress }" ref="progress"></div>
</template>

<script>
import { mapState } from "vuex";
import throttle from "lodash.throttle";
import debounce from "lodash.debounce";

export default {
  name: "ScrollProgress",
  data() {
    return {
      scrollProgress: 0,
      animation: { cancel: () => {} },
      parentElement: document,
    };
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
      this.parentElement.addEventListener("scroll", this.onScroll);
    },
    detachEvents() {
      this.parentElement.removeEventListener("scroll", this.onScroll);
    },
    onScroll: throttle(function () {
      const p = this.parentElement == document ? document.documentElement : this.parentElement;
      this.scrollProgress = p.scrollTop / (p.scrollHeight - p.clientHeight);
      this.animation.cancel();
      this.animation = this.$refs.progress.animate(
        { opacity: [1, 0] },
        {
          duration: 500,
          delay: 2000,
          fill: "both",
          easing: "ease-out",
        }
      );
    }, 150),
  },
};
</script>
<style scoped lang="scss">
.progress {
  background: #00d1b2;
  background-repeat: no-repeat;
  position: fixed;
  height: 4px;
  z-index: 2;
  left: 0;
  right: 0;
  top: 0;
  transform: scaleX(var(--progress));
  transform-origin: left;
  transition: transform 0.3s ease;
  will-change: transform;
}
</style>
