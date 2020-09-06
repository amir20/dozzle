<template>
  <div class="scroll-progress">
    <svg width="100" height="100" viewBox="0 0 100 100" :class="{ indeterminate }">
      <circle r="44" cx="50" cy="50" :style="{ '--progress': scrollProgress }" />
    </svg>
    <div class="is-overlay columns is-vcentered is-centered has-text-weight-light">
      <template v-if="indeterminate">
        <div class="column is-narrow is-paddingless is-size-2">&#8734;</div>
      </template>
      <template v-else>
        <span class="column is-narrow is-paddingless is-size-2">
          {{ Math.ceil(scrollProgress * 100) }}
        </span>
        <span class="column is-narrow is-paddingless"> % </span>
      </template>
    </div>
  </div>
</template>

<script>
import { mapGetters } from "vuex";
import throttle from "lodash.throttle";

export default {
  name: "ScrollProgress",
  props: {
    indeterminate: {
      default: false,
      type: Boolean,
    },
    autoHide: {
      default: true,
      type: Boolean,
    },
  },
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
    indeterminate() {
      this.$nextTick(() => this.onScroll());
    },
  },
  computed: {
    ...mapGetters(["activeContainers"]),
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
      if (this.autoHide) {
        this.animation = this.$el.animate(
          { opacity: [1, 0] },
          {
            duration: 500,
            delay: 2000,
            fill: "both",
            easing: "ease-out",
          }
        );
      }
    },
  },
};
</script>
<style scoped lang="scss">
.scroll-progress {
  display: inline-block;
  position: relative;

  svg {
    filter: drop-shadow(0px 1px 1px rgba(0, 0, 0, 0.2));
    margin-top: 5px;
    &.indeterminate {
      animation: 2s linear infinite svg-animation;

      circle {
        animation: 1.4s ease-in-out infinite both circle-animation;
      }
    }
    circle {
      fill: var(--scheme-main-ter);
      fill-opacity: 0.8;
      transition: stroke-dashoffset 250ms ease-out;
      transform: rotate(-90deg);
      transform-origin: 50% 50%;
      stroke: var(--primary-color);
      stroke-dashoffset: calc(276.32px - var(--progress) * 276.32px);
      stroke-dasharray: 276.32px 276.32px;
      stroke-linecap: round;
      stroke-width: 3;
      will-change: stroke-dashoffset;
    }
  }
}

@keyframes svg-animation {
  0% {
    transform: rotateZ(0deg);
  }
  100% {
    transform: rotateZ(360deg);
  }
}

@keyframes circle-animation {
  0%,
  25% {
    stroke-dashoffset: 275px;
    transform: rotate(0);
  }
  50%,
  75% {
    stroke-dashoffset: 70px;
    transform: rotate(45deg);
  }

  100% {
    stroke-dashoffset: 275px;
    transform: rotate(360deg);
  }
}
</style>
