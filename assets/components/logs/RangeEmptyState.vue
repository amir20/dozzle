<template>
  <EmptyState :title :hint data-testid="range-empty">
    <template #icon><mdi:clock-outline class="size-6" /></template>
    <template #actions v-if="nearest?.before || nearest?.after">
      <button v-if="nearest.before" type="button" class="btn btn-sm" @click="go(nearest.before)">
        <mdi:arrow-up class="size-4 opacity-60" />
        {{ $t("time-range.jump-before", { time: relative(nearest.before) }) }}
      </button>
      <button v-if="nearest.after" type="button" class="btn btn-sm" @click="go(nearest.after)">
        <mdi:arrow-down class="size-4 opacity-60" />
        {{ $t("time-range.jump-after", { time: relative(nearest.after) }) }}
      </button>
    </template>
  </EmptyState>
</template>

<script lang="ts" setup>
import type { Container } from "@/models/Container";
import { rangeAround, timeRangeRoute, type TimeRange } from "@/composable/logs/timeRange";

const { container, range } = defineProps<{
  container: Container;
  range: Extract<TimeRange, { kind: "since" | "range" }>;
}>();

const { t } = useI18n();
const router = useRouter();

const window = computed(() =>
  range.kind === "range" ? { from: range.from, to: range.until } : { from: range.since, to: new Date() },
);
const span = computed(() => window.value.to.getTime() - window.value.from.getTime());

// The server answers an empty window with where the nearest lines are, which
// is the one thing an empty view can usefully offer.
const { data: nearest } = useLogHistogram(
  toRef(() => container),
  window,
  10,
);

const dateTime = (d: Date) => d.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
const title = computed(() =>
  range.kind === "range" ? t("time-range.empty-range") : t("time-range.empty-since", { time: dateTime(range.since) }),
);
const hint = computed(() =>
  range.kind === "range"
    ? t("time-range.empty-range-hint", { from: dateTime(range.from), to: dateTime(range.until) })
    : t("time-range.empty-since-hint"),
);

const relative = (d: Date) => toRelativeTime(d, locale.value === "" ? undefined : locale.value);

// The same span as the empty one, centered on the line, so the person lands
// with that line in the middle of a window like the one they asked for.
function go(at: Date) {
  router.push(timeRangeRoute(container.id, rangeAround(at, span.value)));
}
</script>
