<template>
  <div
    class="indeterminate-bar"
    :style="{ '--bar-color': `var(--color-${color})`, opacity: dimming }"
    role="presentation"
  >
    <div class="indeterminate-bar__comet"></div>
  </div>
</template>
<script setup lang="ts">
const { color = "primary", intensity = 1 } = defineProps<{
  color: "primary" | "error" | "secondary";
  /** 0 = idle (nothing is arriving), 1 = busy. Fades the whole strip rather
   * than changing its speed, so the sweep never jumps mid-flight. */
  intensity?: number;
}>();

const dimming = computed(() => 0.35 + 0.65 * Math.min(Math.max(intensity, 0), 1));
</script>

<style scoped>
/* A dim rail is always drawn so the strip reads as a track the light travels
 * along, instead of a loose glow floating over the background. */
.indeterminate-bar {
  position: relative;
  height: 0.25rem;
  overflow: hidden;
  background: color-mix(in oklab, var(--bar-color) 12%, transparent);
  /* Slow enough that a burst of logs reads as the strip lighting up and easing
   * back down, not as a flicker. */
  transition: opacity 900ms ease-out;
}

/* The comet is a third of the track wide with the bright core near its leading
 * edge and a long tail, so it reads as one object moving in one direction. */
.indeterminate-bar__comet {
  position: absolute;
  inset-block: 0;
  left: 0;
  width: 33.333%;
  background: linear-gradient(
    to right,
    transparent 0%,
    color-mix(in oklab, var(--bar-color) 25%, transparent) 45%,
    color-mix(in oklab, var(--bar-color) 70%, transparent) 78%,
    var(--bar-color) 88%,
    color-mix(in oklab, var(--bar-color) 55%, transparent) 96%,
    transparent 100%
  );
  will-change: transform;
  /* Travels off both edges (-105% and 320% of its own width) so it never turns
   * around on screen. The easing is nearly all spent off the right edge, which
   * keeps the speed even across the view and leaves a short beat between
   * passes rather than a hard restart. */
  animation: comet 2.2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

@keyframes comet {
  from {
    transform: translateX(-105%);
  }
  to {
    transform: translateX(320%);
  }
}

@media (prefers-reduced-motion: reduce) {
  .indeterminate-bar {
    transition: none;
  }

  .indeterminate-bar__comet {
    width: 100%;
    transform: none;
    background: linear-gradient(to right, transparent, var(--bar-color), transparent);
    animation: comet-breathe 3s ease-in-out infinite;
  }

  @keyframes comet-breathe {
    0%,
    100% {
      opacity: 0.35;
    }
    50% {
      opacity: 1;
    }
  }
}
</style>
