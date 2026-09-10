<template>
  <div class="w-full overflow-x-auto" v-if="!loading">
    <table class="w-full border-collapse text-sm" v-if="columns.length">
      <thead>
        <tr>
          <th
            v-for="column in columns"
            :key="column"
            class="bg-base-100 border-base-content/15 text-base-content/50 sticky top-0 z-10 border-b px-3 py-2 text-left font-mono text-xs font-medium whitespace-nowrap"
          >
            {{ column }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(row, index) in table" :key="index" class="result-row">
          <td v-for="column in columns" :key="column" class="max-w-md px-3 py-1.5 align-top">
            <span v-if="format(row[column]) === null" class="text-base-content/30 italic">NULL</span>
            <span v-else class="block truncate font-mono" :title="format(row[column]) ?? undefined">{{
              format(row[column])
            }}</span>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="text-base-content/50 flex flex-col items-center gap-2 py-16">
      <ph:database class="size-8 opacity-40" />
      <span>{{ $t("analytics.no_results") }}</span>
    </div>
  </div>
  <table class="w-full border-collapse text-sm" v-else>
    <thead>
      <tr>
        <th v-for="i in 3" :key="i" class="border-base-content/15 border-b px-3 py-2 text-left">
          <div class="bg-base-content/50 h-3 w-20 animate-pulse rounded opacity-50"></div>
        </th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="i in 9" :key="i" class="border-base-content/10 border-b">
        <td v-for="j in 3" :key="j" class="px-3 py-2">
          <div class="bg-base-content/50 h-3 w-20 animate-pulse rounded opacity-20"></div>
        </td>
      </tr>
    </tbody>
  </table>
</template>
<script lang="ts" setup>
import { type Table } from "@apache-arrow/esnext-esm";

const { loading, table } = defineProps<{
  loading: boolean;
  table: Table<Record<string, any>>;
}>();

const columns = computed(() => (table.numRows > 0 ? Object.keys(table.get(0) as Record<string, any>) : []));

function format(value: unknown): string | null {
  if (value === null || value === undefined) return null;
  if (typeof value === "bigint") return value.toString();
  if (typeof value === "object") {
    try {
      return JSON.stringify(value, (_, v) => (typeof v === "bigint" ? v.toString() : v));
    } catch {
      return String(value);
    }
  }
  return String(value);
}
</script>
<style scoped>
@reference "@/main.css";

/* Zebra striping fought the drawer's own surfaces; a hairline plus a hover tint
   reads the same as the field table in LogDetails. */
.result-row {
  @apply border-base-content/10 border-b transition-colors;
}
.result-row:hover {
  background-color: color-mix(in oklab, var(--color-base-content) 6%, transparent);
}
</style>
