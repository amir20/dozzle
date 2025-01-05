<template>
  <table class="table-zebra table-pin-rows table-md table" v-if="!loading">
    <thead>
      <tr>
        <th v-for="column in columns" :key="column">{{ column }}</th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="row in table" :key="row">
        <td v-for="column in columns" :key="column">{{ row[column] }}</td>
      </tr>
    </tbody>
  </table>
  <table class="table-md table animate-pulse" v-else>
    <thead>
      <tr>
        <th v-for="_ in 3">
          <div class="bg-base-content/50 h-4 w-20 animate-pulse opacity-50"></div>
        </th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="_ in 9">
        <td v-for="_ in 3">
          <div class="bg-base-content/50 h-4 w-20 opacity-20"></div>
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
</script>
