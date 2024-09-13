<template>
  <header class="flex items-center gap-4">
    <h1 class="mobile-hidden text-lg">{{ container.name }}</h1>
    <h2 class="text-sm"><DistanceTime :date="container.created" /></h2>
  </header>

  <div class="mt-8 flex flex-col gap-10">{{ table.length }}</div>
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

const results = await conn.query(`SELECT * FROM logs`);

const table = ref(results.toArray());
</script>
<style lang="postcss" scoped></style>
