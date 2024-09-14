<template>
  <aside class="flex flex-col gap-2">
    <header class="flex items-center gap-4">
      <h1 class="mobile-hidden text-lg">{{ container.name }}</h1>
      <h2 class="text-sm"><DistanceTime :date="container.created" /></h2>
    </header>

    <section>
      <textarea v-model="query" class="textarea textarea-primary w-full"></textarea>
    </section>

    <section>Total {{ results.numRows }} records</section>

    <table class="table table-zebra table-pin-rows table-md">
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
  </aside>
</template>

<script setup lang="ts">
import { Container } from "@/models/Container";
const { container } = defineProps<{ container: Container }>();
const query = ref("SELECT * FROM logs");
const debouncedQuery = debouncedRef(query, 1000);

const url = withBase(
  `/api/hosts/${container.host}/containers/${container.id}/logs?stdout=1&stderr=1&everything&jsonOnly`,
);

const response = await fetch(url);
if (!response.ok) {
  console.log("error fetching logs from", url);
  throw new Error(`Failed to fetch logs: ${response.statusText}`);
}

const { db, conn } = await useDuckDB();

await db.registerFileBuffer("logs.json", new Uint8Array(await response.arrayBuffer()));

await conn.query(`CREATE TABLE logs AS SELECT unnest(m) FROM 'logs.json'`);

const results = computedAsync(
  async () => await conn.query<Record<string, any>>(debouncedQuery.value),
  { numRows: 0 },
  {
    onError: (error) => console.error(error),
  },
);

const columns = computed(() =>
  results.value.numRows > 0 ? Object.keys(results.value.get(0) as Record<string, any>) : [],
);
const page = computed(() => (results.value.numRows > 0 ? results.value.slice(0, 20) : []));
</script>
<style lang="postcss" scoped></style>
