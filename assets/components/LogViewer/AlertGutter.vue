<template>
  <!-- Fixed to the scroll container's right edge rather than positioned inside
       it: the container scrolls, and a marker that scrolls away with the
       content is exactly the thing this is meant to replace. -->
  <div
    v-if="ticks.length"
    class="pointer-events-none fixed z-10"
    :style="{ top: `${top}px`, height: `${height}px`, left: `${right - 10}px`, width: '10px' }"
    data-testid="alert-gutter"
  >
    <button
      v-for="tick in ticks"
      :key="tick.alertId"
      type="button"
      class="pointer-events-auto absolute right-0 h-1 rounded-l-xs opacity-70 transition-[width,opacity] hover:opacity-100"
      :class="tick.count > 1 ? 'w-2.5' : 'w-1.5'"
      :style="{ top: `${tick.offset * 100}%` }"
      :data-alert-level="tick.level"
      :title="tick.count > 1 ? `${tick.headline} (+${tick.count - 1})` : tick.headline"
      @click="scrollTo(tick.alertId)"
    />
  </div>
</template>

<script lang="ts" setup>
import { buildTicks, type AlertMeasurement, type GutterTick } from "@/utils/gutter";

const { container } = defineProps<{ container: HTMLElement | undefined }>();

const ticks = ref<GutterTick[]>([]);
const el = computed(() => container);
// Destructured so the template gets auto-unwrapped refs; `bounds.right`
// would be a Ref in the style binding, not a number.
const { top, height, right } = useElementBounding(el);

function measure() {
  const root = container;
  if (!root) {
    ticks.value = [];
    return;
  }

  const rootTop = root.getBoundingClientRect().top;
  const measurements: AlertMeasurement[] = [];

  for (const node of root.querySelectorAll<HTMLElement>("[data-alert-id]")) {
    measurements.push({
      alertId: Number(node.dataset.alertId),
      // Position within the scrollable content, not the viewport: the two
      // differ by exactly the current scroll offset, and only the former is
      // stable as the user scrolls.
      top: node.getBoundingClientRect().top - rootTop + root.scrollTop,
      level: node.dataset.alertLevel ?? "unknown",
      headline: node.dataset.alertHeadline ?? "",
    });
  }

  ticks.value = buildTicks(measurements, root.scrollHeight, root.clientHeight);
}

// Scrollback prepends entries, which shifts every position below it, so this
// re-runs on content change as well as on resize. Debounced because a
// scrollback load mutates the list many times in quick succession.
const remeasure = useDebounceFn(measure, 100);

useMutationObserver(el, remeasure, { childList: true, subtree: true });
useResizeObserver(el, remeasure);
onMounted(measure);

function scrollTo(alertId: number) {
  container?.querySelector(`[data-alert-id="${alertId}"]`)?.scrollIntoView({ behavior: "smooth", block: "center" });
}
</script>

<style scoped>
@reference "@/main.css";
/* data-alert-level, not data-level — see AlertLogItem.vue. A tick carrying
   data-level would be repainted by LogLevel.vue's global !important rules,
   which would silently turn every info tick green. */
[data-alert-level="error"],
[data-alert-level="fatal"],
[data-alert-level="critical"],
[data-alert-level="severe"] {
  @apply bg-error;
}
[data-alert-level="warn"],
[data-alert-level="warning"] {
  @apply bg-warning;
}
[data-alert-level="info"],
[data-alert-level="debug"],
[data-alert-level="trace"],
[data-alert-level="unknown"] {
  @apply bg-base-content/40;
}
</style>
