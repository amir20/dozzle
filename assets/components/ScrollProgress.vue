<template>
  <transition name="fadeout">
    <div class="pointer-events-none relative inline-block" ref="root" v-show="!autoHide || show">
      <svg width="100" height="100" viewBox="0 0 100 100" :class="{ indeterminate }">
        <circle r="44" cx="50" cy="50" class="fill-base-darker stroke-primary" />
      </svg>
      <div class="absolute inset-0 flex items-center justify-center font-light">
        <template v-if="indeterminate">
          <div class="text-4xl">&#8734;</div>
        </template>
        <template v-else-if="!isNaN(scrollProgress)">
          <span class="text-4xl">
            {{ Math.ceil(scrollProgress * 100) }}
          </span>
          <span> % </span>
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

const pinnedLogsStore = usePinnedLogsStore();
const { pinnedLogs } = storeToRefs(pinnedLogsStore);

const scrollElement = ref<HTMLElement | Document>((root.value?.closest("[data-scrolling]") as HTMLElement) ?? document);
const { y: scrollY } = useScroll(scrollElement as Ref<HTMLElement | Document>, { throttle: 100 });
const show = autoResetRef(false, 2000);

onMounted(() => {
  watch(
    pinnedLogs,
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
<style scoped lang="postcss">
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
    fill-opacity: 0.75;
    transition: stroke-dashoffset 250ms ease-out;
    transform: rotate(-90deg);
    transform-origin: 50% 50%;
    stroke-dashoffset: calc(276.32px - v-bind(scrollProgress) * 276.32px);
    stroke-dasharray: 276.32px 276.32px;
    stroke-linecap: round;
    stroke-width: 3;
    will-change: stroke-dashoffset;
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

.fadeout-leave-active {
  @apply transition-opacity;
}

.fadeout-leave-to {
  opacity: 0;
}
</style>
