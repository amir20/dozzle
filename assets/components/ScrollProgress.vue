<template>
  <!--
    Where-am-I-in-time readout for a paused log stream. It floats over the
    stream, so it reads as a panel in the same family as the toasts and
    dropdowns (neutral surface, hairline border, backdrop blur) instead of the
    saturated dial it used to be, and it stays a single short row so it covers
    one line of logs rather than a block of them.
  -->
  <Transition name="scroll-progress">
    <div
      v-if="show"
      class="rounded-box border-base-content/10 bg-base-200 flex items-center gap-2.5 border py-1.5 pr-2.5 pl-3 shadow-md"
    >
      <template v-if="available">
        <RelativeTime :date="date" class="text-base-content/80 text-xs whitespace-nowrap" />
        <div class="bg-base-content/25 size-1 shrink-0 rounded-full" role="presentation"></div>
      </template>

      <div class="w-24 shrink-0" role="presentation">
        <IndeterminateBar v-if="indeterminate" color="primary" />
        <div v-else class="bg-base-content/15 h-1 overflow-hidden rounded-full">
          <div class="bg-primary h-full rounded-full" :style="{ width: `${eased * 100}%` }"></div>
        </div>
      </div>

      <span v-if="available" class="text-base-content/50 w-8 text-right font-mono text-[0.65rem] tabular-nums">
        {{ Math.round(eased * 100) }}%
      </span>
    </div>
  </Transition>
</template>

<script lang="ts" setup>
const {
  indeterminate = false,
  available = true,
  show = true,
  progress,
  date = new Date(),
} = defineProps<{
  /** Draw the sweeping comet instead of a fill: position is unknown right now. */
  indeterminate?: boolean;
  /** False when nothing computes a position (merged views), so the readout
   * drops the date and the percentage rather than showing a stale 100%. */
  available?: boolean;
  /** Shown while the user is moving and for a beat afterwards, the way an
   * overlay scrollbar behaves: an opaque panel sitting over the stream has to
   * earn its place, so it leaves once nobody is reading it. */
  show?: boolean;
  progress: number;
  date?: Date;
}>();

const target = computed(() => Math.min(Math.max(progress, 0), 1));
const eased = ref(target.value);

// The source position only moves when a new row's timestamp crosses the middle
// of the view, so it arrives in visible steps. Gliding towards it on an
// exponential decay (frame-rate independent, and re-aimed rather than
// restarted when the next step lands mid-flight) is what turns those steps
// into one continuous motion. The number and the fill both read `eased`, so
// they can never disagree.
const reducedMotion = usePreferredReducedMotion();
let frame = 0;

watch(target, (value) => {
  if (reducedMotion.value === "reduce") {
    cancelAnimationFrame(frame);
    eased.value = value;
    return;
  }
  if (frame) return;

  let last = performance.now();
  const step = (now: number) => {
    const delta = now - last;
    last = now;
    const remaining = target.value - eased.value;
    if (Math.abs(remaining) < 0.0005) {
      eased.value = target.value;
      frame = 0;
      return;
    }
    eased.value += remaining * (1 - Math.exp(-delta / 90));
    frame = requestAnimationFrame(step);
  };
  frame = requestAnimationFrame(step);
});

onScopeDispose(() => cancelAnimationFrame(frame));
</script>

<style scoped>
.scroll-progress-enter-active {
  transition:
    opacity 200ms ease-out,
    transform 200ms cubic-bezier(0.22, 1, 0.36, 1);
}

/* Leaving is slower than arriving: it should drift out of the way rather than
   blink off the moment scrolling stops. */
.scroll-progress-leave-active {
  transition:
    opacity 800ms ease-out,
    transform 800ms cubic-bezier(0.22, 1, 0.36, 1);
}

.scroll-progress-enter-from,
.scroll-progress-leave-to {
  opacity: 0;
  transform: translateY(-0.5rem);
}

@media (prefers-reduced-motion: reduce) {
  .scroll-progress-enter-active {
    transition: opacity 200ms ease-out;
  }

  .scroll-progress-leave-active {
    transition: opacity 800ms ease-out;
  }

  .scroll-progress-enter-from,
  .scroll-progress-leave-to {
    transform: none;
  }
}
</style>
