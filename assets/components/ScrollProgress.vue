<template>
  <transition name="fadeout">
    <div class="inline-flex flex-col items-end gap-2" ref="root" v-show="!autoHide || show">
      <div class="relative inline-block">
        <svg width="100" height="100" viewBox="0 0 100 100" :class="{ indeterminate }">
          <circle r="44" cx="50" cy="50" class="fill-base-300 stroke-primary" />
        </svg>
        <div class="absolute inset-0 flex items-center justify-center font-light">
          <span class="text-4xl">
            {{ Math.ceil(progress * 100) }}
          </span>
          <span> % </span>
        </div>
      </div>
      <RelativeTime :date="date" class="text-sm whitespace-nowrap" />
    </div>
  </transition>
</template>

<script lang="ts" setup>
const {
  indeterminate = false,
  autoHide = false,
  progress,
  date = new Date(),
} = defineProps<{
  indeterminate?: boolean;
  autoHide?: boolean;
  progress: number;
  date?: Date;
}>();

const show = autoResetRef(false, 2000);

watch(
  () => progress,
  () => {
    if (autoHide) {
      show.value = true;
    }
  },
);
</script>
<style scoped>
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
    stroke-dashoffset: calc(276.32px - v-bind(progress) * 276.32px);
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
