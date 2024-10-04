<template>
  <table class="table table-zebra table-pin-rows table-md" v-if="!loading">
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
  <table class="table table-md animate-pulse" v-else>
    <thead>
      <tr>
        <th v-for="_ in 3">
          <div class="h-4 w-20 animate-pulse bg-base-content/50 opacity-50"></div>
        </th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="_ in 9">
        <td v-for="_ in 3">
          <div class="h-4 w-20 bg-base-content/50 opacity-20"></div>
        </td>
      </tr>
    </tbody>
  </table>
</template>
<script lang="ts" setup>
import { Table } from "@apache-arrow/ts";

const { loading, table } = defineProps<{
  loading: boolean;
  table: Table<Record<string, any>>;
}>();

const columns = computed(() => (table.numRows > 0 ? Object.keys(table.get(0) as Record<string, any>) : []));
</script>
