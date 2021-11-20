<template>
  <div class="scroll-progress" ref="root">
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

<script lang="ts" setup>
import { useContainerStore } from "@/stores/container";
import throttle from "lodash.throttle";
import { storeToRefs } from "pinia";
import { onMounted, onUnmounted, ref, watchPostEffect } from "vue";

const props = defineProps({
  indeterminate: {
    default: false,
    type: Boolean,
  },
  autoHide: {
    default: true,
    type: Boolean,
  },
});

const scrollProgress = ref(0);
const animation = ref({ cancel: () => {} });
const parentElement = ref<Node>(document);
const root = ref<HTMLElement>();
const store = useContainerStore();
const { activeContainers } = storeToRefs(store);
const onScrollThrottled = throttle(onScroll, 150);

function onScroll() {
  const parent = parentElement.value == document ? document.documentElement : (parentElement.value as HTMLElement);
  scrollProgress.value = parent.scrollTop / (parent.scrollHeight - parent.clientHeight);
  animation.value.cancel();
  if (props.autoHide && root.value) {
    animation.value = root.value.animate(
      { opacity: [1, 0] },
      {
        duration: 500,
        delay: 2000,
        fill: "both",
        easing: "ease-out",
      }
    );
  }
}

function attachEvents() {
  parentElement.value = root.value?.closest("[data-scrolling]") || document;
  parentElement.value.addEventListener("scroll", onScrollThrottled);
}

function detachEvents() {
  parentElement.value.removeEventListener("scroll", onScrollThrottled);
}

onMounted(() => {
  attachEvents();
});

onUnmounted(() => {
  detachEvents();
});

watchPostEffect(() => {
  activeContainers.value.length;
  detachEvents();
  attachEvents();
});
</script>
<style scoped lang="scss">
.scroll-progress {
  display: inline-block;
  position: relative;
  pointer-events: none;

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
