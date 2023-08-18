<template>
  <transition name="fade">
    <div class="scroll-progress" ref="root" v-show="!autoHide || show">
      <svg width="100" height="100" viewBox="0 0 100 100" :class="{ indeterminate }">
        <circle r="44" cx="50" cy="50" />
      </svg>
      <div class="is-overlay columns is-vcentered is-centered has-text-weight-light">
        <template v-if="indeterminate">
          <div class="column is-narrow is-paddingless is-size-2">&#8734;</div>
        </template>
        <template v-else-if="!isNaN(scrollProgress)">
          <span class="column is-narrow is-paddingless is-size-2">
            {{ Math.ceil(scrollProgress * 100) }}
          </span>
          <span class="column is-narrow is-paddingless"> % </span>
        </template>
      </div>
    </div>
  </transition>
</template>

<script lang="ts" setup>
const { indeterminate = false, autoHide = false } = defineProps<{
  indeterminate?: boolean;
  autoHide?: boolean;
}>();

const scrollProgress = ref(0);
const root = ref<HTMLElement>();
const store = useContainerStore();
const { activeContainers } = storeToRefs(store);
const scrollElement = ref<HTMLElement | Document>((root.value?.closest("[data-scrolling]") as HTMLElement) ?? document);
const { y: scrollY } = useScroll(scrollElement as Ref<HTMLElement | Document>, { throttle: 100 });
const show = autoResetRef(false, 2000);

onMounted(() => {
  watch(
    activeContainers,
    () => {
      scrollElement.value = (root.value?.closest("[data-scrolling]") as HTMLElement) ?? document;
    },
    { immediate: true, flush: "post" },
  );
});

watchPostEffect(() => {
  const parent =
    scrollElement.value === document
      ? (scrollElement.value as Document).documentElement
      : (scrollElement.value as HTMLElement);
  scrollProgress.value = Math.max(0, Math.min(1, scrollY.value / (parent.scrollHeight - parent.clientHeight)));
  if (autoHide) {
    show.value = true;
  }
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
      stroke-dashoffset: calc(276.32px - v-bind(scrollProgress) * 276.32px);
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

.fade-leave-active {
  transition: opacity 0.2s ease-in-out;
}

.fade-leave-to {
  opacity: 0;
}
</style>
