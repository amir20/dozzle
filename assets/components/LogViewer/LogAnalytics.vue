<template>
  <aside>
    <header class="flex items-center gap-4">
      <h1 class="text-2xl max-md:hidden">{{ container.name }}</h1>
      <h2 class="text-sm"><RelativeTime :date="container.created" /></h2>
    </header>

    <div class="mt-8 flex flex-col gap-2">
      <section>
        <label class="form-control">
          <textarea
            v-model="query"
            class="textarea textarea-primary w-full font-mono text-lg"
            :class="{ 'textarea-error!': error }"
            :disabled="state === 'downloading'"
          ></textarea>
          <div class="mt-2">
            <span class="text-error" v-if="error">{{ error }}</span>
            <span v-else-if="state === 'initializing'">{{ $t("analytics.creating_table") }}</span>
            <span v-else-if="state === 'downloading'">{{
              $t("analytics.downloading", { size: formatBytes(bytes, { decimals: 1 }) })
            }}</span>
            <span v-else-if="evaluating">{{ $t("analytics.evaluating_query") }}</span>
            <span v-else>
              {{ $t("analytics.total_records", { count: results.numRows.toLocaleString() }) }}
              <template v-if="results.numRows > pageLimit">{{
                $t("analytics.showing_first", { count: page.numRows.toLocaleString() })
              }}</template>
            </span>
          </div>
        </label>
      </section>
      <SQLTable :table="page" :loading="evaluating || state !== 'ready'" />
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
const state = ref<"downloading" | "ready" | "initializing">("downloading");
const bytes = ref(0);

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

onMounted(async () => {
  try {
    state.value = "downloading";

    const reader = response.body?.getReader();
    if (!reader) throw new Error("No reader available from stream");

    const chunks: Uint8Array[] = [];
    bytes.value = 0;

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      chunks.push(value);
      bytes.value += value.length;
    }

    const arrayBuffer = new Uint8Array(bytes.value);
    let position = 0;
    for (const chunk of chunks) {
      arrayBuffer.set(chunk, position);
      position += chunk.length;
    }

    await db.registerFileBuffer("logs.json", arrayBuffer);

    state.value = "initializing";
    await conn.query(
      `CREATE TABLE logs AS SELECT unnest(m) FROM read_json('logs.json', ignore_errors = true, format = 'newline_delimited')`,
    );

    state.value = "ready";
  } catch (e) {
    console.error(e);
    if (e instanceof Error) {
      error.value = e.message;
    }
  }
});

const results = computedAsync(
  async () => {
    if (state.value === "ready") {
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

whenever(evaluating, () => {
  error.value = null;
  state.value = "ready";
});
const page = computed(() =>
  results.value.numRows > pageLimit ? results.value.slice(0, pageLimit) : results.value,
) as unknown as ComputedRef<Table<Record<string, any>>>;
</script>
<style scoped></style>
