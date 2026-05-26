<template>
  <div class="w-full overflow-x-auto" v-if="!loading">
    <table class="table-zebra table-pin-rows table-md table" v-if="columns.length">
      <thead>
        <tr>
          <th v-for="column in columns" :key="column" class="font-mono">{{ column }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(row, index) in table" :key="index">
          <td v-for="column in columns" :key="column" class="max-w-md align-top">
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
  <table class="table-md table" v-else>
    <thead>
      <tr>
        <th v-for="i in 3" :key="i">
          <div class="bg-base-content/50 h-4 w-20 animate-pulse opacity-50"></div>
        </th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="i in 9" :key="i">
        <td v-for="j in 3" :key="j">
          <div class="bg-base-content/50 h-4 w-20 animate-pulse opacity-20"></div>
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
