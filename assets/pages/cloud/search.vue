<template>
  <PageWithLinks>
    <section>
      <!-- Header -->
      <div class="mb-5 flex items-center gap-3">
        <h2 class="text-lg font-semibold">{{ $t("cloud-search.results-page-title") }}</h2>
        <span v-if="committedQuery" class="text-base-content/70 font-mono text-sm">"{{ committedQuery }}"</span>
        <span v-if="cloudSearch.available.value" class="status-pill status-pill-primary ml-auto">
          <mdi:flash class="size-3" /> {{ $t("cloud-search.hero-pill-indexed") }}
        </span>
      </div>

      <!-- Status line -->
      <div class="text-base-content/70 mb-3 flex h-5 items-center gap-2 text-xs">
        <template v-if="cloudSearch.loading.value">
          <span class="loading loading-spinner loading-xs"></span>
          <span>{{ $t("cloud-search.searching") }}</span>
        </template>
        <template v-else-if="cloudSearch.error.value">
          <mdi:alert-circle-outline class="text-error size-3.5" />
          <span>{{ $t("cloud-search.search-failed") }}</span>
        </template>
        <template v-else-if="committedQuery && hits.length === 0">
          <span>{{ $t("cloud-search.no-results") }}</span>
        </template>
        <template v-else-if="!committedQuery">
          <span>{{ $t("cloud-search.search-empty-prompt") }}</span>
        </template>
        <template v-else>
          <span class="font-mono">{{ $t("cloud-search.hits-count", { n: hits.length }) }}</span>
          <span class="text-base-content/50">{{ $t("cloud-search.window-suffix") }}</span>
        </template>
      </div>

      <!-- Results table — matches the visual style of ContainerTable -->
      <div v-if="hits.length" class="rounded-box border-base-content/10 overflow-x-auto border">
        <table class="table-md md:table-lg table-zebra table">
          <thead>
            <tr>
              <th class="text-base-content/60 w-44 text-xs font-medium tracking-wider uppercase">
                {{ $t("cloud-search.col-time") }}
              </th>
              <th class="text-base-content/60 w-20 text-xs font-medium tracking-wider uppercase">
                {{ $t("cloud-search.col-level") }}
              </th>
              <th class="text-base-content/60 w-1 text-xs font-medium tracking-wider uppercase">
                {{ $t("cloud-search.col-container") }}
              </th>
              <th class="text-base-content/60 text-xs font-medium tracking-wider uppercase">
                {{ $t("cloud-search.col-message") }}
              </th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(hit, i) in hits"
              :key="`${hit.containerId}-${hit.ts}-${hit.logId ?? 0}-${i}`"
              class="hover:bg-primary/5 transition-colors"
              :class="{ 'cursor-pointer': isLive(hit) }"
              @click="isLive(hit) && openContainer(hit)"
            >
              <td class="text-base-content/70 font-mono text-xs whitespace-nowrap tabular-nums">
                {{ formatTs(hit.ts) }}
              </td>
              <td>
                <span class="status-pill" :class="levelPillClass(hit.level)">{{ hit.level || "info" }}</span>
              </td>
              <td class="whitespace-nowrap">
                <span class="inline-flex items-center gap-2">
                  <span :class="isLive(hit) ? 'text-base-content' : 'text-base-content/60'">
                    {{ hit.containerName }}
                  </span>
                  <span
                    v-if="!isLive(hit)"
                    :title="$t('cloud-search.container-removed')"
                    class="status-pill status-pill-neutral"
                  >
                    {{ $t("cloud-search.container-removed-pill") }}
                  </span>
                </span>
              </td>
              <td>
                <JsonFormatted
                  v-if="isJson(hit.message)"
                  :value="hit.message"
                  :highlight="committedQuery"
                  class="text-xs"
                />
                <span v-else class="font-mono text-xs" v-html="highlight(hit.message, committedQuery)"></span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div
        v-if="hits.length && (cloudSearch.hasMore.value || cloudSearch.loadingMore.value)"
        class="text-base-content/60 mt-4 flex h-10 items-center justify-center text-xs"
      >
        <span v-if="cloudSearch.loadingMore.value" class="loading loading-spinner loading-xs"></span>
      </div>

      <!-- Cloud-not-available state -->
      <div
        v-if="!cloudSearch.available.value && committedQuery"
        class="bg-base-200 border-base-content/10 rounded-box border p-8 text-center"
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
import { useCloudConfig } from "@/composable/cloudConfig";
import { useCloudLogSearch, type CloudLogHit } from "@/composable/cloudLogSearch";

const route = useRoute();
const router = useRouter();

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
useInfiniteScroll(document, () => cloudSearch.loadMore(), {
  distance: 200,
  canLoadMore: () => cloudSearch.hasMore.value && !cloudSearch.loadingMore.value,
});

watch(
  () => route.query.q,
  (q) => {
    committedQuery.value = readQ(q);
  },
);

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
  // Match Dozzle's permanent-link route: /container/:id/time/:datetime?logId=...
  // hit.ts is unix nanoseconds; convert to ms then ISO 8601 with millis.
  const datetime = new Date(hit.ts / 1e6).toISOString();
  const query: Record<string, string> = {};
  if (hit.logId !== undefined && hit.logId !== 0) {
    // logId pinpoints the exact line; the historical-logs view scrolls to it.
    query.logId = String(hit.logId);
  }
  if (committedQuery.value) {
    query.q = committedQuery.value;
  }
  router.push({
    name: "/container/[id].time.[datetime]",
    params: { id: hit.containerId, datetime },
    query,
  });
}
</script>
