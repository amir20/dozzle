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
          <template v-if="hovered[s.name]">
            {{ formatRange(hovered[s.name]!) }} · {{ $t("analytics.rows", hovered[s.name]!.count) }}
          </template>
          <template v-else>n={{ s.count.toLocaleString() }}</template>
        </span>
      </div>

      <dl class="mt-2 grid grid-cols-3 gap-3 sm:grid-cols-5">
        <div v-for="stat in statsOf(s)" :key="stat.label" class="min-w-0">
          <dt class="text-base-content/50 text-[0.7rem] font-semibold tracking-wider uppercase">{{ stat.label }}</dt>
          <dd class="truncate font-mono text-lg font-semibold tabular-nums" :title="String(stat.value)">
            {{ formatNumber(stat.value) }}
          </dd>
        </div>
      </dl>

      <Histogram
        class="mt-3"
        :bins="s.bins"
        :format="formatNumber"
        @hover-bin="(bin: HistogramBin) => (hovered[s.name] = bin)"
        @hover-end="delete hovered[s.name]"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { DataType, type Table } from "@apache-arrow/esnext-esm";
import { distribution, type Distribution, type HistogramBin } from "./histogramBins";

const { table } = defineProps<{
  table: Table<Record<string, any>>;
}>();

type Series = Distribution & { name: string };

const { t } = useI18n();
const hovered = reactive<Record<string, HistogramBin>>({});

// A chart only when every column is a number: the moment a text column rides
// along, the result is a listing, and bars beside it would say nothing.
// DECIMAL never reaches here as such: useDuckDB casts it to DOUBLE.
//
// A histogram rather than a series: a charted field has no natural x-axis (the
// query has no ORDER BY and rows are not evenly spaced in time), so what the
// result can honestly show is the shape of the values, tail included.
const series = computed<Series[]>(() => {
  const fields = table.schema.fields;
  if (table.numRows < 2 || fields.length === 0) return [];
  if (!fields.every((f) => DataType.isInt(f.type) || DataType.isFloat(f.type))) return [];

  return fields
    .map((field) => summarize(field.name, DataType.isInt(field.type)))
    .filter((s): s is Series => s !== null);
});

function summarize(name: string, integer: boolean): Series | null {
  const column = table.getChild(name)!;
  const values = new Float64Array(column.length);
  let count = 0;
  for (const raw of column) {
    if (raw === null || raw === undefined) continue;
    const value = Number(raw);
    if (Number.isFinite(value)) values[count++] = value;
  }

  // Every row NULL (say TRY_CAST over a text field): nothing was measured, and
  // zeros for every stat would claim otherwise.
  const result = distribution(values.subarray(0, count), integer);
  return result && { ...result, name };
}

watch(
  () => table,
  () => {
    for (const key of Object.keys(hovered)) delete hovered[key];
  },
);

function statsOf(s: Series) {
  return [
    { label: t("analytics.min"), value: s.min },
    { label: t("analytics.p50"), value: s.p50 },
    { label: t("analytics.avg"), value: s.avg },
    { label: t("analytics.p95"), value: s.p95 },
    { label: t("analytics.max"), value: s.max },
  ];
}

function formatNumber(value: number) {
  return value.toLocaleString(undefined, { maximumFractionDigits: 2 });
}

function formatRange(bin: HistogramBin) {
  return bin.start === bin.end ? formatNumber(bin.start) : `${formatNumber(bin.start)} – ${formatNumber(bin.end)}`;
}
</script>
