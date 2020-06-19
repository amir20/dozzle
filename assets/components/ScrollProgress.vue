<template>
  <div class="progress" :style="{ '--progress': scrollProgress }" ref="progress"></div>
</template>

<script>
import throttle from "lodash.throttle";

export default {
  name: "ScrollProgress",
  data() {
    return {
      scrollProgress: 0,
    };
  },
  mounted() {
    const onScroll = throttle(() => {
      const p = document.documentElement;
      this.scrollProgress = p.scrollTop / (p.scrollHeight - p.clientHeight);
      this.$refs.progress.animate([{ opacity: 1 }, { opacity: 0 }], {
        duration: 300,
        fill: "both",
        delay: 2000,
      });
    }, 150);
    document.addEventListener("scroll", onScroll);
    this.$once("hook:beforeDestroy", () => document.removeEventListener("scroll", onScroll));
  },
};
</script>
<style scoped lang="scss">
.progress {
  background: #00d1b2;
  background-repeat: no-repeat;
  position: fixed;
  width: 100%;
  height: 4px;
  z-index: 1;
  left: 0;
  transform: scaleX(var(--progress));
  transform-origin: left;
  transition: transform 0.3s ease;
  will-change: transform;
}
</style>
