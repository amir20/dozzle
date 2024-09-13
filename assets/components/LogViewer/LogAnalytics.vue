<template>
  <header class="flex items-center gap-4">
    <h1 class="mobile-hidden text-lg">{{ container.name }}</h1>
    <h2 class="text-sm"><DistanceTime :date="container.created" /></h2>
  </header>

  <div class="mt-8 flex flex-col gap-10">
    {{ table.numRows }} of rows

    <table class="table">
      <thead>
        <tr>
          <th v-for="column in columns" :key="column">{{ column }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="row in page" :key="row">
          <td v-for="column in columns" :key="column">{{ row[column] }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { Container } from "@/models/Container";
const { container } = defineProps<{ container: Container }>();
const url = withBase(
  `/api/hosts/${container.host}/containers/${container.id}/logs?stdout=1&stderr=1&everything&jsonOnly`,
);
const { db, conn } = await useDuckDB();

const response = await fetch(url);

if (!response.ok) {
  console.log("error fetching logs from", url);
  throw new Error(`Failed to fetch logs: ${response.statusText}`);
}

await db.registerFileBuffer("logs.json", new Uint8Array(await response.arrayBuffer()));

await conn.query(`CREATE TABLE logs AS SELECT unnest(m) FROM 'logs.json'`);

const results = await conn.query<Record<string, any>>(`SELECT * FROM logs`);

const table = ref(results);
const page = computed(() => results.slice(0, 10));
const rows = shallowRef(results.toArray());
const columns = computed(() => {
  if (rows.value.length === 0) return [];
  return Object.keys(rows.value[0]);
});
</script>
<style lang="postcss" scoped></style>
