<template>
  <aside>
    <header class="flex items-center gap-4">
      <h1 class="text-2xl max-md:hidden">{{ container.name }}</h1>
      <h2 class="text-sm"><DistanceTime :date="container.created" /></h2>
    </header>

    <div class="mt-8 flex flex-col gap-2">
      <section>
        <label class="form-control">
          <textarea
            v-model="query"
            class="textarea textarea-primary w-full font-mono text-lg"
            :class="{ 'textarea-error': error }"
          ></textarea>
          <div class="label max-h-48 overflow-y-auto pr-2">
            <span class="label-text-alt text-error" v-if="error">{{ error }}</span>
            <span class="label-text-alt" v-else>
              Total {{ results.numRows }} records
              <template v-if="results.numRows > pageLimit"> . Showing first {{ page.numRows }}. </template>
            </span>
          </div>
        </label>
      </section>
      <SQLTable :table="page" :loading="evaluating || !isReady" />
    </div>
  </aside>
</template>

<script setup lang="ts">
import { Container } from "@/models/Container";
import { type Table } from "@apache-arrow/esnext-esm";
const { container } = defineProps<{ container: Container }>();
const query = ref("SELECT * FROM logs LIMIT 100");
const error = ref<string | null>(null);
const debouncedQuery = debouncedRef(query, 500);
const evaluating = ref(false);
const pageLimit = 1000;

const url = withBase(
  `/api/hosts/${container.host}/containers/${container.id}/logs?stdout=1&stderr=1&everything&jsonOnly`,
);

const [{ useDuckDB }, response] = await Promise.all([import(`@/composable/duckdb`), fetch(url)]);

if (!response.ok) {
  console.log("error fetching logs from", url);
  throw new Error(`Failed to fetch logs: ${response.statusText}`);
}

const { db, conn } = await useDuckDB();
const empty = await conn.query<Record<string, any>>(`SELECT 1 LIMIT 0`);

const { isReady } = useAsyncState(
  async () => {
    await db.registerFileBuffer("logs.json", new Uint8Array(await response.arrayBuffer()));
    await conn.query(
      `CREATE TABLE logs AS SELECT unnest(m) FROM read_json('logs.json', ignore_errors = true, format = 'newline_delimited')`,
    );
  },
  undefined,
  {
    onError: (e) => {
      console.error(e);
      if (e instanceof Error) {
        error.value = e.message;
      }
    },
  },
);

const results = computedAsync(
  async () => {
    if (isReady.value) {
      return await conn.query<Record<string, any>>(debouncedQuery.value);
    } else {
      return empty;
    }
  },
  empty,
  {
    onError: (e) => {
      console.error(e);
      if (e instanceof Error) {
        error.value = e.message;
      }
    },
    evaluating,
  },
);

whenever(evaluating, () => (error.value = null));
const page = computed(() =>
  results.value.numRows > pageLimit ? results.value.slice(0, pageLimit) : results.value,
) as unknown as ComputedRef<Table<Record<string, any>>>;
</script>
<style scoped></style>
