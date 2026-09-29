<template>
  <!-- pr-24 keeps the title clear of the drawer's maximize/close buttons, which
       float over this slot. -->
  <header class="border-base-content/10 flex flex-wrap items-center gap-x-3 gap-y-1 border-b pr-24 pb-4">
    <ph:file-sql class="text-primary size-6 shrink-0" />
    <h1 class="text-base font-semibold">{{ $t("analytics.title") }}</h1>
    <h2 class="text-base-content/55 flex min-w-0 items-center text-xs">
      <span class="truncate">{{ container.name }}</span>
      <span class="px-1">&middot;</span>
      <RelativeTime :date="container.created" />
    </h2>
  </header>

  <div class="mt-5 flex flex-col gap-6 pb-8">
    <section class="flex flex-col gap-2">
      <div class="flex items-center gap-3">
        <div class="field-label">{{ $t("analytics.query") }}</div>
        <div class="text-base-content/40 ml-auto flex items-center gap-1.5 text-xs">
          <KeyShortcut char="&crarr;" />
          {{ $t("analytics.run") }}
        </div>
      </div>

      <div
        class="bg-base-200 focus-within:border-primary/60 min-h-24 rounded-md border px-3 py-1"
        :class="error ? 'border-error/50' : 'border-base-content/10'"
      >
        <div ref="editorEl" class="w-full" :aria-label="$t('analytics.title')"></div>
      </div>

      <!-- Binder errors name the offending column and clause, so they wrap
           rather than truncate: the useful half is usually at the end. -->
      <p
        v-if="error"
        class="border-error/30 bg-error/10 text-error rounded-md border px-3 py-2 font-mono text-xs leading-relaxed whitespace-pre-wrap"
      >
        {{ error }}
      </p>
    </section>

    <section class="flex flex-col gap-3" v-if="state === 'ready' && columns.length">
      <div class="flex flex-wrap items-center gap-x-3 gap-y-1.5">
        <div class="field-label">{{ $t("analytics.examples") }}</div>
        <button v-for="ex in examples" :key="ex.key" class="chip" @click="applyExample(ex.sql)">
          {{ $t(ex.key, ex.params ?? {}) }}
        </button>
      </div>

      <details class="group">
        <summary class="text-base-content/50 hover:text-base-content/80 flex w-fit cursor-pointer items-center gap-1">
          <ph:caret-right class="size-3 transition-transform group-open:rotate-90" />
          <span class="field-label">{{ $t("analytics.columns") }}</span>
          <span class="text-xs opacity-60">{{ columns.length }}</span>
        </summary>
        <div class="mt-2 flex max-h-40 flex-wrap gap-1.5 overflow-y-auto">
          <button
            v-for="col in columns"
            :key="col.name"
            class="chip font-mono"
            :title="col.type"
            @click="insertColumn(col.name)"
          >
            {{ col.name }}
          </button>
        </div>
      </details>
    </section>

    <section class="flex flex-col gap-2">
      <div class="flex items-center gap-3">
        <div class="field-label shrink-0">{{ $t("analytics.results") }}</div>

        <p class="text-base-content/45 min-w-0 truncate text-xs">
          <template v-if="state === 'loading'"></template>
          <span class="inline-flex items-center gap-2" v-else-if="state === 'initializing'">
            <span class="loading loading-spinner loading-xs"></span>{{ $t("analytics.creating_table") }}
          </span>
          <span class="inline-flex items-center gap-2" v-else-if="evaluating">
            <span class="loading loading-spinner loading-xs"></span>{{ $t("analytics.evaluating_query") }}
          </span>
          <template v-else>
            {{ $t("analytics.total_records", { count: results.numRows.toLocaleString() }) }}
            <template v-if="results.numRows > pageLimit">{{
              $t("analytics.showing_first", { count: page.numRows.toLocaleString() })
            }}</template>
          </template>
        </p>

        <Popover
          class="ml-auto shrink-0"
          placement="bottom-end"
          panel-class="bg-base-200 rounded-box w-44 p-2 shadow-sm"
          v-if="canExport"
        >
          <template #trigger>
            <button type="button" class="btn btn-xs btn-ghost gap-1">
              <ph:download-simple class="size-4" />
              {{ $t("analytics.export") }}
            </button>
          </template>
          <ul class="menu w-full p-0">
            <li>
              <a class="cursor-pointer whitespace-nowrap" @click="exportResults('csv')">{{
                $t("analytics.export_csv")
              }}</a>
            </li>
            <li>
              <a class="cursor-pointer whitespace-nowrap" @click="exportResults('json')">{{
                $t("analytics.export_json")
              }}</a>
            </li>
          </ul>
        </Popover>
      </div>

      <SQLChart :table="resultTable" v-if="state === 'ready' && !evaluating" />

      <!-- No height cap: the drawer is the one scroller, and a capped box inside it
           trapped the wheel halfway down the page. -->
      <div class="border-base-content/10 overflow-hidden rounded-md border">
        <!-- The engine is ~5 MB on first open and cached after, so the one slow wait
             gets a real bar instead of a spinner. The logs have no known length. -->
        <div v-if="state === 'loading' && !error" class="divide-base-content/10 divide-y text-sm">
          <div class="flex flex-col gap-2 p-4">
            <div class="flex items-baseline justify-between gap-2">
              <span class="text-base-content/60">{{ $t("analytics.loading_engine") }}</span>
              <span class="font-mono font-semibold">{{ Math.round(engineProgress * 100) }}%</span>
            </div>
            <div class="bg-base-content/10 h-1.5 w-full overflow-hidden rounded-full">
              <div
                class="bg-primary h-full rounded-full transition-[width] duration-500 motion-reduce:transition-none"
                :style="{ width: `${engineProgress * 100}%` }"
              ></div>
            </div>
          </div>
          <div class="flex items-center justify-between gap-2 p-4">
            <span class="text-base-content/60">{{ $t("analytics.fetching_logs") }}</span>
            <span class="flex items-center gap-2 font-mono">
              {{ formatBytes(bytes, { decimals: 1 }) }}
              <mdi:check v-if="logsDone" class="text-success size-4" />
              <span v-else class="loading loading-spinner loading-xs opacity-60"></span>
            </span>
          </div>
        </div>
        <SQLTable v-else :table="page" :loading="!error && (evaluating || state !== 'ready')" />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { Container } from "@/models/Container";
import { Table } from "@apache-arrow/esnext-esm";
import type { AsyncDuckDBConnection } from "@duckdb/duckdb-wasm";

const { container, initialQuery } = defineProps<{
  container: Container;
  // Opens with this query instead of the default, e.g. charting one field from LogDetails.
  initialQuery?: string;
}>();
const defaultQuery = "SELECT * FROM logs LIMIT 100";
const query = ref(initialQuery ?? defaultQuery);
const error = ref<string | null>(null);
const evaluating = ref(false);
const pageLimit = 1000;
const state = ref<"loading" | "initializing" | "ready">("loading");
const bytes = ref(0);
const logsDone = ref(false);
const engineProgress = ref(0);
const columns = ref<{ name: string; type: string }[]>([]);
const editorEl = ref<HTMLElement>();

const { setValue, insertAtCursor } = useSQLEditorField(editorEl, {
  placeholder: defaultQuery,
  // Read once: the editor owns its content after mount, and re-seeding it on every model
  // change would fight the user's cursor.
  initialValue: query.value,
  // Read lazily so completions pick up the schema the moment DESCRIBE returns.
  getColumns: () => columns.value,
  onRun: () => run(),
  onChange: (v) => (query.value = v),
});

const runQuery = ref(query.value);
// An edit that changes the query is a run the user asked for. The first run when
// the view opens is not, and neither is the debounce catching up after run().
watchDebounced(
  query,
  (v) => {
    // Only a query that actually runs counts; before the data is loaded none does.
    if (v !== runQuery.value && state.value === "ready") trackUsage("logs.sql");
    runQuery.value = v;
  },
  { debounce: 500 },
);

const url = withBase(
  `/api/hosts/${container.host}/containers/${container.id}/logs?stdout=1&stderr=1&everything&jsonOnly`,
);

// Nothing here is awaited in setup, so the drawer renders at once and shows progress
// instead of sitting on the Suspense spinner. Both downloads start now and run side by
// side: reading the logs only after the engine was up had left them mostly serial.
const engine = useDuckDB((fraction) => (engineProgress.value = fraction));
const abort = new AbortController();
onUnmounted(() => abort.abort());

let conn: AsyncDuckDBConnection | undefined;
const empty = new Table<Record<string, any>>();

async function fetchLogs(): Promise<Uint8Array> {
  const response = await fetch(url, { signal: abort.signal });
  if (!response.ok) throw new Error(`Failed to fetch logs: ${response.statusText}`);

  const reader = response.body?.getReader();
  if (!reader) throw new Error("No reader available from stream");

  const chunks: Uint8Array[] = [];
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(value);
    bytes.value += value.length;
  }
  logsDone.value = true;

  const buffer = new Uint8Array(bytes.value);
  let position = 0;
  for (const chunk of chunks) {
    buffer.set(chunk, position);
    position += chunk.length;
  }
  return buffer;
}

(async () => {
  try {
    const [duck, buffer] = await Promise.all([engine, fetchLogs()]);
    const { db } = duck;
    conn = duck.conn;

    await db.registerFileBuffer("logs.json", buffer);

    state.value = "initializing";
    await conn.query(
      `CREATE TABLE logs AS SELECT unnest(m) FROM read_json('logs.json', ignore_errors = true, format = 'newline_delimited', map_inference_threshold = -1)`,
    );

    const described = await conn.query<{ column_name: any; column_type: any }>(`DESCRIBE logs`);
    columns.value = described.toArray().map((row) => ({
      name: String(row.column_name),
      type: String(row.column_type),
    }));

    state.value = "ready";
  } catch (e) {
    // Closing the drawer aborts the logs download; that is not an error to show.
    if (abort.signal.aborted) return;
    console.error(e);
    if (e instanceof Error) {
      error.value = e.message;
    }
  }
})();

const examples = computed(() => {
  const names = columns.value.map((c) => c.name);
  const pick = ["level", "severity", "lvl", "status"].find((c) => names.includes(c)) ?? names[0];
  const list: { key: string; sql: string; params?: Record<string, string> }[] = [
    { key: "analytics.example_all", sql: "SELECT * FROM logs LIMIT 100" },
    { key: "analytics.example_count", sql: "SELECT count(*) AS total FROM logs" },
  ];
  if (pick) {
    list.push({
      key: "analytics.example_group",
      params: { column: pick },
      sql: `SELECT "${pick}", count(*) AS count FROM logs GROUP BY "${pick}" ORDER BY count DESC`,
    });
  }
  return list;
});

function run() {
  // The same query again changes nothing, so nothing runs and nothing counts.
  if (state.value !== "ready" || query.value === runQuery.value) return;
  trackUsage("logs.sql");
  runQuery.value = query.value;
}

function applyExample(sql: string) {
  setValue(sql);
  nextTick(run);
}

function insertColumn(name: string) {
  // Matches what completion applies: bare when it is a legal identifier, quoted otherwise.
  insertAtCursor(/^[A-Za-z_][A-Za-z0-9_]*$/.test(name) ? name : `"${name}"`);
}

const results = computedAsync(
  async () => {
    if (state.value === "ready" && conn) {
      return await conn.query<Record<string, any>>(runQuery.value);
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

// Only the error: state is the loader's to set. The editor is live while the engine
// loads, and flipping to ready from here showed an empty result before any table existed.
// A failed load is never retried, so typing must not wipe its error.
whenever(evaluating, () => {
  if (state.value === "ready") error.value = null;
});
// Stats and chart cover every row the query returned, not just the page the table shows.
const resultTable = results as unknown as Ref<Table<Record<string, any>>>;
const page = computed(() =>
  results.value.numRows > pageLimit ? results.value.slice(0, pageLimit) : results.value,
) as unknown as ComputedRef<Table<Record<string, any>>>;

const canExport = computed(() => state.value === "ready" && !evaluating.value && results.value.numRows > 0);

function stringify(value: unknown): string {
  if (value === null || value === undefined) return "";
  if (typeof value === "bigint") return value.toString();
  if (typeof value === "object") return JSON.stringify(value, (_, v) => (typeof v === "bigint" ? v.toString() : v));
  return String(value);
}

function toCSV(table: Table<Record<string, any>>, columns: string[]): string {
  const escape = (value: string) => (/[",\n\r]/.test(value) ? `"${value.replaceAll('"', '""')}"` : value);
  const lines = [columns.map(escape).join(",")];
  for (const row of table) {
    lines.push(columns.map((column) => escape(stringify((row as Record<string, any>)[column]))).join(","));
  }
  return lines.join("\n");
}

function toJSON(table: Table<Record<string, any>>, columns: string[]): string {
  const rows = [];
  for (const row of table) {
    const record: Record<string, unknown> = {};
    for (const column of columns) {
      const value = (row as Record<string, any>)[column];
      record[column] = typeof value === "bigint" ? value.toString() : value;
    }
    rows.push(record);
  }
  return JSON.stringify(rows, null, 2);
}

function exportResults(format: "csv" | "json") {
  const table = results.value as unknown as Table<Record<string, any>>;
  if (table.numRows === 0) return;

  const columns = Object.keys(table.get(0) as Record<string, any>);
  const content = format === "csv" ? toCSV(table, columns) : toJSON(table, columns);
  const type = format === "csv" ? "text/csv;charset=utf-8" : "application/json";
  const name = container.name.replace(/[^\w.-]+/g, "-");

  const url = URL.createObjectURL(new Blob([content], { type }));
  const link = document.createElement("a");
  link.href = url;
  link.download = `${name}-query.${format}`;
  link.click();
  URL.revokeObjectURL(url);
  (document.activeElement as HTMLElement | null)?.blur();
}
</script>
<style scoped>
@reference "@/main.css";

/* Same label style as LogDetails, so both log drawers give the eye one "this is
   a label, not content" cue. */
.field-label {
  @apply text-base-content/50 text-[0.7rem] font-semibold tracking-wider uppercase;
}

/* Examples and column names are both "click to put this in the query", so they
   share one affordance instead of two different daisyUI badge variants. */
.chip {
  @apply border-base-content/15 text-base-content/80 cursor-pointer rounded border px-2 py-0.5 text-xs transition-colors;
}
.chip:hover {
  @apply border-primary/50 text-primary;
}
</style>
