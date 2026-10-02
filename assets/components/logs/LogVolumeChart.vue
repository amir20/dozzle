<template>
  <div class="flex flex-col gap-1">
    <div
      ref="strip"
      class="border-base-content/15 relative flex touch-none items-end gap-px border-b select-none"
      :class="[heightClass, selectable ? 'cursor-crosshair' : '']"
      @pointerdown="down"
      @pointermove="move"
      @pointerup="up"
      @pointercancel="dragging = undefined"
    >
      <template v-if="data">
        <!-- A column is the full height so the whole strip reads the pointer, and its
             title is the only tooltip: a few hundred of these must stay cheap. -->
        <div
          v-for="bar in bars"
          :key="bar.start"
          class="flex h-full min-w-0 flex-1 flex-col justify-end"
          :class="{ 'opacity-40': bar.outside }"
          :title="bar.title"
        >
          <div v-if="bar.unknown" class="unscanned h-full rounded-t-sm"></div>
          <template v-else>
            <div class="bg-base-content/30 rounded-t-sm" :style="{ height: bar.okHeight }"></div>
            <div class="bg-error" :style="{ height: bar.errorHeight }"></div>
          </template>
        </div>
      </template>
      <div v-else class="bg-base-content/5 size-full animate-pulse rounded-sm"></div>

      <div
        v-if="overlay"
        class="border-primary/60 bg-primary/5 pointer-events-none absolute -top-1 -bottom-0.5 rounded border"
        :style="overlay"
      ></div>
      <span
        v-if="data && isEmptyHistogram(data)"
        class="text-base-content/40 pointer-events-none absolute inset-0 flex items-center justify-center text-xs"
      >
        {{ $t("time-range.empty-window") }}
      </span>
    </div>
    <div class="text-base-content/40 flex justify-between font-mono text-xs tabular-nums" v-if="data">
      <span v-for="tick in ticks" :key="tick">{{ tick }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { isEmptyHistogram, type LogHistogram } from "@/composable/logs/logHistogram";

const {
  data,
  selection,
  selectable = false,
  heightClass = "h-10",
} = defineProps<{
  data?: LogHistogram;
  /** The range the view shows; bars outside it are dimmed and it is outlined. */
  selection?: { from: Date; to: Date };
  selectable?: boolean;
  heightClass?: string;
}>();

const select = defineEmit<[from: Date, to: Date]>();
const { t } = useI18n();

const strip = useTemplateRef<HTMLElement>("strip");
const dragging = ref<{ start: number; end: number }>();

const span = computed(() => (data ? data.total.length * data.width : 0));

const bars = computed(() => {
  if (!data) return [];
  const peak = Math.max(1, ...data.total);
  const timeFormat = timeFormatFor(data.width);
  return data.total.map((total, i) => {
    const start = data.start.getTime() + i * data.width;
    const end = start + data.width;
    const errors = data.errors[i];
    const unknown = data.scannedFrom !== undefined && end <= data.scannedFrom.getTime();
    // Never thinner than a sliver, so a bucket with one line is still visible
    // next to one with thousands.
    const height = (n: number) => (n === 0 ? "0" : `max(2px, ${(n / peak) * 100}%)`);
    return {
      start,
      unknown,
      outside: selection !== undefined && (end <= selection.from.getTime() || start >= selection.to.getTime()),
      okHeight: height(total - errors),
      errorHeight: height(errors),
      title: unknown
        ? `${timeFormat.format(start)} · ${t("time-range.not-scanned")}`
        : `${timeFormat.format(start)} – ${timeFormat.format(end)} · ${t("time-range.bucket", { lines: total.toLocaleString(), errors: errors.toLocaleString() })}`,
    };
  });
});

const ticks = computed(() => {
  if (!data) return [];
  const format = timeFormatFor(data.width);
  const start = data.start.getTime();
  return [0, 0.25, 0.5, 0.75, 1].map((f) => format.format(start + f * span.value));
});

function timeFormatFor(width: number) {
  return new Intl.DateTimeFormat(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: width < 60_000 ? "2-digit" : undefined,
    month: width >= 60 * 60_000 ? "short" : undefined,
    day: width >= 60 * 60_000 ? "numeric" : undefined,
  });
}

function percent(from: number, to: number) {
  if (!data || span.value === 0) return undefined;
  const start = data.start.getTime();
  const left = Math.min(100, Math.max(0, ((from - start) / span.value) * 100));
  const right = Math.min(100, Math.max(0, ((to - start) / span.value) * 100));
  if (right <= left) return undefined;
  return { left: `${left}%`, width: `${right - left}%` };
}

const overlay = computed(() => {
  if (dragging.value && data) {
    const a = Math.min(dragging.value.start, dragging.value.end);
    const b = Math.max(dragging.value.start, dragging.value.end) + 1;
    const start = data.start.getTime();
    return percent(start + a * data.width, start + b * data.width);
  }
  return selection ? percent(selection.from.getTime(), selection.to.getTime()) : undefined;
});

function bucketAt(e: PointerEvent) {
  const rect = strip.value!.getBoundingClientRect();
  const fraction = Math.min(0.9999, Math.max(0, (e.clientX - rect.left) / rect.width));
  return Math.floor(fraction * (data?.total.length ?? 0));
}

function down(e: PointerEvent) {
  if (!selectable || !data) return;
  strip.value!.setPointerCapture(e.pointerId);
  const i = bucketAt(e);
  dragging.value = { start: i, end: i };
}

function move(e: PointerEvent) {
  if (dragging.value) dragging.value = { ...dragging.value, end: bucketAt(e) };
}

function up() {
  if (!dragging.value || !data) return;
  const a = Math.min(dragging.value.start, dragging.value.end);
  const b = Math.max(dragging.value.start, dragging.value.end) + 1;
  dragging.value = undefined;
  const start = data.start.getTime();
  select(new Date(start + a * data.width), new Date(start + b * data.width));
}
</script>

<style scoped>
.unscanned {
  background: repeating-linear-gradient(
    135deg,
    color-mix(in oklab, var(--color-base-content) 12%, transparent) 0 2px,
    transparent 2px 5px
  );
}
</style>
