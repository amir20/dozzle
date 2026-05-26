<template>
  <aside class="flex flex-col gap-5 pb-8">
    <header class="flex items-center gap-3 pr-8">
      <ph:file-sql class="text-primary size-7 shrink-0" />
      <div class="flex min-w-0 flex-col">
        <h1 class="text-xl leading-tight font-semibold">{{ $t("analytics.title") }}</h1>
        <p class="text-base-content/60 flex items-center gap-1.5 text-sm">
          <span class="truncate">{{ container.name }}</span>
          <span class="opacity-40">·</span>
          <RelativeTime :date="container.created" />
        </p>
      </div>
    </header>

    <section class="flex flex-col gap-2">
      <textarea
        ref="queryEl"
        v-model="query"
        class="textarea textarea-primary w-full resize-y font-mono text-sm leading-relaxed"
        :class="{ 'textarea-error!': error }"
        :disabled="state !== 'ready'"
        rows="3"
        spellcheck="false"
        autocapitalize="off"
        autocomplete="off"
        :aria-label="$t('analytics.title')"
        @keydown.meta.enter.prevent="run"
        @keydown.ctrl.enter.prevent="run"
      ></textarea>

      <div class="flex min-h-6 items-center text-sm">
        <div class="min-w-0 flex-1 truncate">
          <span class="text-error" v-if="error">{{ error }}</span>
          <span class="text-base-content/60 inline-flex items-center gap-2" v-else-if="state === 'initializing'">
            <span class="loading loading-spinner loading-xs"></span>{{ $t("analytics.creating_table") }}
          </span>
          <span class="text-base-content/60 inline-flex items-center gap-2" v-else-if="state === 'downloading'">
            <span class="loading loading-spinner loading-xs"></span
            >{{ $t("analytics.downloading", { size: formatBytes(bytes, { decimals: 1 }) }) }}
          </span>
          <span class="text-base-content/60 inline-flex items-center gap-2" v-else-if="evaluating">
            <span class="loading loading-spinner loading-xs"></span>{{ $t("analytics.evaluating_query") }}
          </span>
          <span class="text-base-content/60" v-else>
            {{ $t("analytics.total_records", { count: results.numRows.toLocaleString() }) }}
            <template v-if="results.numRows > pageLimit">{{
              $t("analytics.showing_first", { count: page.numRows.toLocaleString() })
            }}</template>
          </span>
        </div>
      </div>
    </section>

    <section v-if="state === 'ready' && columns.length" class="flex flex-col gap-2 text-xs">
      <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
        <span class="text-base-content/50 font-medium">{{ $t("analytics.examples") }}</span>
        <button
          v-for="ex in examples"
          :key="ex.key"
          class="badge badge-sm badge-outline hover:border-primary hover:text-primary cursor-pointer"
          @click="applyExample(ex.sql)"
        >
          {{ $t(ex.key, ex.params ?? {}) }}
        </button>
      </div>

      <details class="group">
        <summary
          class="text-base-content/50 hover:text-base-content/80 flex w-fit cursor-pointer items-center gap-1 font-medium select-none"
        >
          <ph:caret-right class="size-3 transition-transform group-open:rotate-90" />
          {{ $t("analytics.columns") }}
          <span class="opacity-60">{{ columns.length }}</span>
        </summary>
        <div class="mt-2 flex max-h-40 flex-wrap gap-1.5 overflow-y-auto">
          <button
            v-for="col in columns"
            :key="col.name"
            class="badge badge-sm badge-ghost hover:border-primary hover:text-primary cursor-pointer font-mono"
            :title="col.type"
            @click="insertColumn(col.name)"
          >
            {{ col.name }}
          </button>
        </div>
      </details>
    </section>

    <SQLTable :table="page" :loading="evaluating || state !== 'ready'" />
  </aside>
</template>

<script setup lang="ts">
import { Container } from "@/models/Container";
import { type Table } from "@apache-arrow/esnext-esm";

const { container } = defineProps<{ container: Container }>();
const query = ref("SELECT * FROM logs LIMIT 100");
const error = ref<string | null>(null);
const evaluating = ref(false);
const pageLimit = 1000;
const state = ref<"downloading" | "ready" | "initializing">("downloading");
const bytes = ref(0);
const columns = ref<{ name: string; type: string }[]>([]);
const queryEl = useTemplateRef<HTMLTextAreaElement>("queryEl");

const runQuery = ref(query.value);
watchDebounced(query, (v) => (runQuery.value = v), { debounce: 500 });

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
      `CREATE TABLE logs AS SELECT unnest(m) FROM read_json('logs.json', ignore_errors = true, format = 'newline_delimited', map_inference_threshold = -1)`,
    );

    const described = await conn.query<{ column_name: any; column_type: any }>(`DESCRIBE logs`);
    columns.value = described.toArray().map((row) => ({
      name: String(row.column_name),
      type: String(row.column_type),
    }));

    state.value = "ready";
  } catch (e) {
    console.error(e);
    if (e instanceof Error) {
      error.value = e.message;
    }
  }
});

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
  if (state.value !== "ready") return;
  runQuery.value = query.value;
}

function applyExample(sql: string) {
  query.value = sql;
  nextTick(run);
}

function insertColumn(name: string) {
  const text = `"${name}"`;
  const el = queryEl.value;
  if (!el) {
    query.value += text;
    return;
  }
  const start = el.selectionStart ?? query.value.length;
  const end = el.selectionEnd ?? start;
  query.value = query.value.slice(0, start) + text + query.value.slice(end);
  nextTick(() => {
    el.focus();
    const pos = start + text.length;
    el.setSelectionRange(pos, pos);
  });
}

const results = computedAsync(
  async () => {
    if (state.value === "ready") {
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

whenever(evaluating, () => {
  error.value = null;
  state.value = "ready";
});
const page = computed(() =>
  results.value.numRows > pageLimit ? results.value.slice(0, pageLimit) : results.value,
) as unknown as ComputedRef<Table<Record<string, any>>>;
</script>
<style scoped></style>
