<template>
  <PageWithLinks>
    <!-- The top bar's search box is desktop-only, and it is the only place the query shows. -->
    <SearchInline class="md:hidden" />
    <section class="flex flex-col gap-4">
      <header class="flex items-start gap-3">
        <div class="min-w-0">
          <h2 class="text-2xl font-bold">{{ $t("cloud-search.results-page-title") }}</h2>
          <p class="text-base-content/60 mt-1 truncate text-sm">
            <span v-if="committedQuery" class="text-base-content font-mono">"{{ committedQuery }}"</span>
            {{ $t("cloud-search.window-suffix") }}
          </p>
        </div>
        <span v-if="cloudSearch.available.value" class="status-pill status-pill-primary ml-auto shrink-0">
          <mdi:flash class="size-3" /> {{ $t("cloud-search.hero-pill-indexed") }}
        </span>
      </header>

      <!-- Status line. One row tall in every state so the results never jump under it. -->
      <div class="text-base-content/60 flex h-5 items-center gap-2 text-xs">
        <template v-if="cloudSearch.loading.value">
          <span class="loading loading-spinner loading-xs"></span>
          <span>{{ $t("cloud-search.searching") }}</span>
        </template>
        <template v-else-if="cloudSearch.error.value">
          <mdi:alert-circle-outline class="text-error size-3.5" />
          <span>{{ $t("cloud-search.search-failed") }}</span>
          <span v-if="cloudSearch.errorDetail.value" class="text-base-content/60">{{
            cloudSearch.errorDetail.value
          }}</span>
        </template>
        <template v-else-if="committedQuery && hits.length === 0">
          <span>{{ $t("cloud-search.no-results") }}</span>
        </template>
        <template v-else-if="!committedQuery">
          <span>{{ $t("cloud-search.search-empty-prompt") }}</span>
        </template>
        <template v-else>
          <span class="text-base-content font-mono font-semibold">{{
            $t("cloud-search.hits-count", { n: hitCount })
          }}</span>
          <template v-if="cloudSearch.tookMs.value !== null">
            <span class="text-base-content/40">·</span>
            <span class="font-mono">{{ formatDuration(cloudSearch.tookMs.value) }}</span>
          </template>
        </template>
      </div>

      <SearchVolume
        v-if="hits.length && cloudSearch.volume.value.length && cloudSearch.stepNs.value > 0"
        :buckets="cloudSearch.volume.value"
        :step-ns="cloudSearch.stepNs.value"
        class="border-base-content/15 bg-base-200/40 rounded-lg border p-4"
      />

      <!-- Narrowing what is already loaded, so it is instant and costs Cloud nothing. -->
      <div v-if="hits.length && (levelCounts.length > 1 || containerCounts.length > 1)" class="flex flex-wrap gap-1.5">
        <button
          v-for="[level, count] in levelCounts"
          :key="`level-${level}`"
          type="button"
          class="btn btn-xs rounded-full font-normal"
          :class="{ 'btn-active border-base-content/30': levels.has(level) }"
          :aria-pressed="levels.has(level)"
          @click="toggle(levels, level)"
        >
          <span class="status" :class="levelDotClass(level)"></span>
          {{ level }}
          <span class="text-base-content/40 font-mono">{{ count }}</span>
        </button>
        <span v-if="levelCounts.length > 1 && containerCounts.length > 1" class="bg-base-content/10 mx-1 w-px"></span>
        <button
          v-for="[name, count] in containerCounts"
          :key="`container-${name}`"
          type="button"
          class="btn btn-xs rounded-full font-normal"
          :class="{ 'btn-active border-base-content/30': containers.has(name) }"
          :aria-pressed="containers.has(name)"
          @click="toggle(containers, name)"
        >
          {{ name }}
          <span class="text-base-content/40 font-mono">{{ count }}</span>
        </button>
      </div>

      <ul
        v-if="visibleHits.length"
        class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border"
      >
        <li v-for="(hit, i) in visibleHits" :key="`${keyOf(hit)}-${i}`">
          <button
            type="button"
            class="hover:bg-base-300 grid w-full grid-cols-[auto_auto_1fr] items-center gap-x-3 gap-y-1 px-4 py-2 text-left transition-colors md:grid-cols-[8.5rem_4.5rem_minmax(0,12rem)_minmax(0,1fr)_auto]"
            :aria-expanded="expanded === keyOf(hit)"
            @click="expanded = expanded === keyOf(hit) ? null : keyOf(hit)"
          >
            <span class="text-base-content/60 font-mono text-xs whitespace-nowrap tabular-nums">
              {{ formatTs(hit.ts) }}
            </span>
            <span class="status-pill justify-self-start" :class="levelPillClass(hit.level)">{{ levelOf(hit) }}</span>
            <span class="truncate text-sm" :class="isLive(hit) ? 'text-base-content' : 'text-base-content/50'">
              {{ hit.containerName }}
            </span>
            <!-- A structured line leads with what it says; the fields that qualify it trail, muted. -->
            <span class="col-span-3 truncate font-mono text-xs md:col-span-1">
              <template v-if="summaryOf(hit).rest">
                <span v-html="highlight(summaryOf(hit).text, committedQuery)"></span>
                <JsonFormatted
                  class="ml-2 opacity-50"
                  :value="summaryOf(hit).rest"
                  :highlight="committedQuery"
                  :block="false"
                />
              </template>
              <JsonFormatted
                v-else-if="isJson(hit.message)"
                :value="hit.message"
                :highlight="committedQuery"
                :block="false"
              />
              <span v-else v-html="highlight(hit.message, committedQuery)"></span>
            </span>
            <mdi:chevron-down
              class="hidden size-4 opacity-40 transition-transform md:block"
              :class="{ 'rotate-180': expanded === keyOf(hit) }"
            />
          </button>
          <div v-if="expanded === keyOf(hit)" class="bg-base-100/60 flex flex-col gap-3 px-4 py-3">
            <div class="max-h-96 overflow-auto font-mono text-xs">
              <JsonFormatted v-if="isJson(hit.message)" :value="hit.message" :highlight="committedQuery" />
              <span v-else class="break-all whitespace-pre-wrap" v-html="highlight(hit.message, committedQuery)"></span>
            </div>
            <div class="flex items-center gap-3">
              <button v-if="isLive(hit)" type="button" class="btn btn-primary btn-sm" @click="openContainer(hit)">
                <material-symbols:eye-tracking class="size-4" />
                {{ $t("action.see-in-context") }}
              </button>
              <span v-else :title="$t('cloud-search.container-removed')" class="status-pill status-pill-neutral">
                {{ $t("cloud-search.container-removed-pill") }}
              </span>
              <span class="text-base-content/40 font-mono text-xs">{{ hit.containerId.slice(0, 12) }}</span>
            </div>
          </div>
        </li>
      </ul>

      <div
        v-if="hits.length && (cloudSearch.hasMore.value || cloudSearch.loadingMore.value)"
        class="text-base-content/60 flex h-10 items-center justify-center text-xs"
      >
        <span v-if="cloudSearch.loadingMore.value" class="loading loading-spinner loading-xs"></span>
      </div>

      <!-- Cloud-not-available state -->
      <div
        v-if="!cloudSearch.available.value && committedQuery"
        class="border-base-content/15 bg-base-200/40 rounded-lg border p-8 text-center"
      >
        <mdi:cloud-off-outline class="text-base-content/40 mx-auto mb-3 size-10" />
        <p class="text-base-content/80 text-sm">
          {{
            cloudConfig?.linked ? $t("cloud-search.enable-streaming-to-search") : $t("cloud-search.connect-to-enable")
          }}
        </p>
        <RouterLink to="/settings/cloud" class="btn btn-primary btn-sm mt-4">
          {{ $t("cloud-search.cta-settings") }}
        </RouterLink>
      </div>
    </section>
  </PageWithLinks>
</template>

<script lang="ts" setup>
import { useCloudConfig } from "@/composable/cloud/cloudConfig";
import { useCloudLogSearch, type CloudLogHit } from "@/composable/cloud/cloudLogSearch";
import { messageKeyOf } from "@/composable/logs/similarSearch";

const route = useRoute();
const { jumpTo } = useLogJump();

function readQ(q: unknown): string {
  return typeof q === "string" ? q : "";
}

const committedQuery = ref(readQ(route.query.q));

const { cloudConfig } = useCloudConfig();
const cloudSearch = useCloudLogSearch(committedQuery);
const hits = computed<CloudLogHit[]>(() => cloudSearch.results.value);

// Look up containers in the live store so we can mark hits whose containers
// have been removed (or never existed for this Dozzle instance) as
// non-clickable. Reactive — if a container is removed mid-session, the
// corresponding row updates instantly.
const containerStore = useContainerStore();
const liveIds = computed(() => new Set(Object.keys(containerStore.allContainersById)));
function isLive(hit: CloudLogHit): boolean {
  return liveIds.value.has(hit.containerId);
}

// Infinite scroll: VueUse fires loadMore when the page is scrolled within
// 200px of the bottom. canLoadMore short-circuits both during a fetch and
// when the server reports no more pages, so we don't double-fire.
const { reset: recheckScroll } = useInfiniteScroll(document, () => cloudSearch.loadMore(), {
  distance: 200,
  canLoadMore: () => cloudSearch.hasMore.value && !cloudSearch.loadingMore.value,
});

watch(
  () => route.query.q,
  (q) => {
    committedQuery.value = readQ(q);
  },
);

// A new query is a new result set: filters and the open row belonged to the old one.
const levels = ref(new Set<string>());
const containers = ref(new Set<string>());
const expanded = ref<string | null>(null);
watch(committedQuery, () => {
  levels.value = new Set();
  containers.value = new Set();
  expanded.value = null;
});

function toggle(set: Set<string>, value: string) {
  if (set.has(value)) set.delete(value);
  else set.add(value);
}

function countBy(key: (hit: CloudLogHit) => string): [string, number][] {
  const counts = new Map<string, number>();
  for (const hit of hits.value) counts.set(key(hit), (counts.get(key(hit)) ?? 0) + 1);
  return [...counts].sort((a, b) => b[1] - a[1]);
}
const levelCounts = computed(() => countBy(levelOf));
const containerCounts = computed(() => countBy((hit) => hit.containerName));

const visibleHits = computed(() =>
  hits.value.filter(
    (hit) =>
      (levels.value.size === 0 || levels.value.has(levelOf(hit))) &&
      (containers.value.size === 0 || containers.value.has(hit.containerName)),
  ),
);

// The scroll watcher only looks again when the page scrolls. A chip that leaves too few
// rows to scroll would otherwise strand the next page, so ask it to look after each change.
watch(() => visibleHits.value.length, recheckScroll);

function keyOf(hit: CloudLogHit): string {
  return `${hit.containerId}-${hit.ts}-${hit.logId ?? 0}`;
}

function levelOf(hit: CloudLogHit): string {
  return (hit.level || "info").toLowerCase();
}

// The histogram counts every match in the window, so with it the total is exact. Without
// it (an older Cloud, or a count that timed out) only the loaded pages are known, and a
// page with more behind it reads as "20+" rather than claiming 20 is the whole answer.
const hitCount = computed(() => {
  const counted = cloudSearch.volume.value.reduce((sum, b) => sum + b.count, 0);
  if (counted >= hits.value.length && counted > 0) return counted.toLocaleString();
  return cloudSearch.hasMore.value ? `${hits.value.length}+` : String(hits.value.length);
});

function formatDuration(ms: number): string {
  return ms < 1000 ? `${ms} ms` : `${(ms / 1000).toFixed(1)} s`;
}

function levelDotClass(level: string): string {
  switch (level) {
    case "error":
    case "fatal":
      return "status-error";
    case "warn":
    case "warning":
      return "status-warning";
    case "info":
      return "status-info";
    default:
      return "status-neutral";
  }
}

function formatTs(ns: number): string {
  const d = new Date(ns / 1e6);
  const date = d.toLocaleDateString([], { month: "short", day: "numeric" });
  const time = d.toLocaleTimeString([], { hour12: false }) + "." + String(d.getMilliseconds()).padStart(3, "0");
  return `${date} ${time}`;
}

function levelPillClass(level: string): string {
  switch ((level || "").toLowerCase()) {
    case "error":
    case "fatal":
      return "status-pill-error";
    case "warn":
    case "warning":
      return "status-pill-warning";
    case "info":
      return "status-pill-primary";
    default:
      return "status-pill-neutral";
  }
}

// Safe with v-html: escapeHtml runs first, then <mark> tags are added against
// a regex anchored on the (already-escaped) needle. Don't drop the escape
// thinking it's redundant — the message comes from indexed log content.
function highlight(message: string, q: string): string {
  if (!q) return escapeHtml(message);
  const pattern = q.replace(/[-/\\^$*+?.()|[\]{}]/g, "\\$&");
  const re = new RegExp(`(${pattern})`, "gi");
  return escapeHtml(message).replace(re, '<mark class="bg-warning text-warning-content rounded px-0.5">$1</mark>');
}

function escapeHtml(s: string): string {
  return s.replace(
    /[&<>"']/g,
    (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c] as string,
  );
}

// Parsed once per hit, not once per render of every row.
const summaries = new WeakMap<CloudLogHit, { text: string; rest?: Record<string, unknown> }>();
function summaryOf(hit: CloudLogHit) {
  let summary = summaries.get(hit);
  if (!summary) {
    summary = { text: hit.message };
    if (isJson(hit.message)) {
      const fields = JSON.parse(hit.message.trim()) as Record<string, unknown>;
      const key = Array.isArray(fields) ? undefined : messageKeyOf(fields);
      if (key) {
        const { [key]: text, ...rest } = fields;
        summary = { text: text as string, rest };
      }
    }
    summaries.set(hit, summary);
  }
  return summary;
}

function isJson(message: string): boolean {
  const trimmed = message.trim();
  if (!trimmed.startsWith("{") && !trimmed.startsWith("[")) return false;
  try {
    const parsed = JSON.parse(trimmed);
    return parsed !== null && typeof parsed === "object";
  } catch {
    return false;
  }
}

function openContainer(hit: CloudLogHit) {
  // hit.ts is unix nanoseconds.
  jumpTo({
    containerId: hit.containerId,
    date: new Date(hit.ts / 1e6),
    logId: hit.logId,
    query: committedQuery.value,
  });
}
</script>
