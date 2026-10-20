<template>
  <div class="flex flex-col gap-1.5" @mouseleave="hovered = null">
    <div class="flex h-20 items-end gap-px">
      <!-- A column is the full height so the whole strip is the hover target, not just
           the painted bar: a bucket holding one line is a few pixels tall. -->
      <div
        v-for="(column, i) in columns"
        :key="column.start"
        class="flex h-full min-w-0 flex-1 flex-col-reverse rounded-t-sm transition-colors"
        :class="{ 'bg-base-content/5': hovered === i }"
        data-testid="volume-column"
        @mouseenter="hovered = i"
        @touchstart.passive="hovered = i"
      >
        <template v-if="column.total">
          <div class="bg-error shrink-0" :style="{ height: share(column.error) }"></div>
          <div class="bg-warning shrink-0" :style="{ height: share(column.warn) }"></div>
          <div class="bg-base-content/30 shrink-0 rounded-t-sm" :style="{ height: share(column.other) }"></div>
        </template>
        <div v-else class="bg-base-content/10 h-px shrink-0"></div>
      </div>
    </div>
    <!-- One line under the chart: the hovered bucket while there is one, otherwise the
         window's ends with what each colour adds up to. -->
    <div class="text-base-content/50 flex items-center gap-3 font-mono text-xs tabular-nums">
      <template v-if="hovered !== null && columns[hovered]">
        <span class="text-base-content">{{ format(columns[hovered].start) }}</span>
        <Band
          v-for="band in bandsOf(columns[hovered])"
          :key="band.key"
          :dot="band.dot"
          :name="band.name"
          :count="band.count"
        />
      </template>
      <template v-else>
        <span>{{ format(columns[0].start) }}</span>
        <Band v-for="band in bandsOf(totals)" :key="band.key" :dot="band.dot" :name="band.name" :count="band.count" />
        <span class="ml-auto">{{ format(columns[columns.length - 1].start) }}</span>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { volumeColumns, type VolumeBucket, type VolumeColumn } from "@/composable/cloud/searchVolume";

const { buckets, stepNs } = defineProps<{ buckets: VolumeBucket[]; stepNs: number }>();

const columns = computed(() => volumeColumns(buckets, stepNs, Date.now()));
const hovered = ref<number | null>(null);
watch(columns, () => (hovered.value = null));

const peak = computed(() => Math.max(1, ...columns.value.map((c) => c.total)));
// Never thinner than a sliver, so one error in a bucket of thousands still shows.
const share = (count: number) => (count ? `max(2px, ${(count / peak.value) * 100}%)` : "0");

const totals = computed(() =>
  columns.value.reduce(
    (sum, c) => ({ ...sum, error: sum.error + c.error, warn: sum.warn + c.warn, other: sum.other + c.other }),
    { start: 0, error: 0, warn: 0, other: 0, total: 0 } as VolumeColumn,
  ),
);

// The neutral band is named by the levels actually in it rather than by a made-up word.
const otherLevels = computed(() => {
  const names = new Set(buckets.map((b) => b.level.toLowerCase() || "unknown"));
  for (const severe of ["error", "fatal", "warn", "warning"]) names.delete(severe);
  return [...names].join(", ");
});

function bandsOf(column: VolumeColumn) {
  return [
    { key: "error", dot: "bg-error", name: "error", count: column.error },
    { key: "warn", dot: "bg-warning", name: "warn", count: column.warn },
    { key: "other", dot: "bg-base-content/30", name: otherLevels.value, count: column.other },
  ].filter((band) => band.count > 0);
}

// A bucket a day or wider is named by its date; anything narrower needs the hour.
function format(ms: number): string {
  const d = new Date(ms);
  const date = d.toLocaleDateString([], { month: "short", day: "numeric" });
  return stepNs >= 86_400e9 ? date : `${date} ${d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}`;
}

const Band = (props: { dot: string; name: string; count: number }) =>
  h("span", { class: "inline-flex items-center gap-1" }, [
    h("span", { class: `size-2 rounded-full ${props.dot}` }),
    h("span", { class: "text-base-content/60" }, props.name),
    h("span", { class: "text-base-content font-semibold" }, props.count.toLocaleString()),
  ]);
</script>
