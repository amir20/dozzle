<template>
  <div class="flex flex-col gap-3" v-if="series.length">
    <div
      v-for="s in series"
      :key="s.name"
      class="border-base-content/15 bg-base-200/40 rounded-lg border p-4"
      data-testid="sql-chart"
    >
      <div class="flex items-baseline gap-3">
        <span class="truncate font-mono text-sm font-semibold" :title="s.name">{{ s.name }}</span>
        <span class="text-base-content/40 ml-auto shrink-0 font-mono text-xs tabular-nums">
          <template v-if="hovered[s.name] !== undefined">{{ formatNumber(hovered[s.name]!) }}</template>
          <template v-else>n={{ s.count.toLocaleString() }}</template>
        </span>
      </div>

      <dl class="mt-2 grid grid-cols-3 gap-3">
        <div v-for="stat in statsOf(s)" :key="stat.label" class="min-w-0">
          <dt class="text-base-content/50 text-[0.7rem] font-semibold tracking-wider uppercase">{{ stat.label }}</dt>
          <dd class="truncate font-mono text-lg font-semibold tabular-nums" :title="String(stat.value)">
            {{ formatNumber(stat.value) }}
          </dd>
        </div>
      </dl>

      <!-- Keyed on the table so a new result remounts the chart: BarChart only
           patches its last bar per tick and never notices a wholesale swap. -->
      <BarChart
        :key="chartKey"
        class="mt-3 h-24"
        bar-class="text-primary"
        :chart-data="s.points"
        @hover-value="(value: number) => (hovered[s.name] = value)"
        @hover-end="delete hovered[s.name]"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { DataType, type Table } from "@apache-arrow/esnext-esm";
import type { BarDataPoint } from "./BarChart.vue";

const { table } = defineProps<{
  table: Table<Record<string, any>>;
}>();

interface Series {
  name: string;
  points: BarDataPoint[];
  count: number;
  min: number;
  max: number;
  avg: number;
}

const { t } = useI18n();
const hovered = reactive<Record<string, number>>({});
const chartKey = ref(0);

// Enough for the widest drawer: BarChart averages down to what fits anyway, so
// handing it more only costs memory and a longer re-bucket.
const MAX_POINTS = 1000;

// A chart only when every column is a number: the moment a text column rides
// along, the result is a listing, and bars beside it would say nothing.
// DECIMAL never reaches here as such: useDuckDB casts it to DOUBLE.
const series = computed<Series[]>(() => {
  const fields = table.schema.fields;
  if (table.numRows < 2 || fields.length === 0) return [];
  if (!fields.every((f) => DataType.isInt(f.type) || DataType.isFloat(f.type))) return [];

  return fields.map((field) => summarize(field.name)).filter((s): s is Series => s !== null);
});

// One pass per column: stats over every row, points pre-averaged into at most
// MAX_POINTS buckets so a large result never lands in BarChart whole.
function summarize(name: string): Series | null {
  const column = table.getChild(name)!;
  const bucketSize = Math.max(1, Math.ceil(column.length / MAX_POINTS));

  let count = 0;
  let min = Infinity;
  let max = -Infinity;
  let sum = 0;
  const averages: number[] = [];
  let bucketSum = 0;
  let bucketCount = 0;

  let index = 0;
  for (const raw of column) {
    if (raw !== null && raw !== undefined) {
      const value = Number(raw);
      if (Number.isFinite(value)) {
        count++;
        sum += value;
        if (value < min) min = value;
        if (value > max) max = value;
        bucketSum += value;
        bucketCount++;
      }
    }
    index++;
    if (index % bucketSize === 0 || index === column.length) {
      if (bucketCount > 0) averages.push(bucketSum / bucketCount);
      bucketSum = 0;
      bucketCount = 0;
    }
  }

  // Every row NULL (say TRY_CAST over a text field): nothing was measured, and
  // zeros for min, avg and max would claim otherwise.
  if (count === 0) return null;

  // Bars grow from zero, so a series that dips below it is lifted by its
  // minimum. The hover still reads the real value.
  const offset = min < 0 ? -min : 0;
  return {
    name,
    points: averages.map((value) => ({ value, percent: value + offset })),
    count,
    min,
    max,
    avg: sum / count,
  };
}

watch(
  () => table,
  () => {
    chartKey.value++;
    for (const key of Object.keys(hovered)) delete hovered[key];
  },
);

function statsOf(s: Series) {
  return [
    { label: t("analytics.min"), value: s.min },
    { label: t("analytics.avg"), value: s.avg },
    { label: t("analytics.max"), value: s.max },
  ];
}

function formatNumber(value: number) {
  return value.toLocaleString(undefined, { maximumFractionDigits: 2 });
}
</script>
