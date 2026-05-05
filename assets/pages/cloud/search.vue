<template>
  <PageWithLinks>
    <section>
      <!-- Header -->
      <div class="mb-6">
        <div class="flex items-baseline gap-3">
          <h2 class="text-lg font-semibold">{{ $t("cloud-search.results-page-title") }}</h2>
          <template v-if="committedQuery">
            <span class="text-base-content/30">/</span>
            <span class="text-base-content font-mono text-sm">"{{ committedQuery }}"</span>
          </template>
          <span
            v-if="cloudSearch.available.value"
            class="bg-primary/15 text-primary border-primary/30 ml-auto inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs"
          >
            <mdi:flash class="size-3" /> {{ $t("cloud-search.hero-pill-indexed") }}
          </span>
        </div>

        <!-- Status line -->
        <div class="text-base-content/50 mt-3 flex items-center gap-2 text-xs">
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
          </template>
        </div>
      </div>

      <!-- Results table — matches the visual style of ContainerTable -->
      <div v-if="hits.length" class="rounded-box border-base-content/10 overflow-x-auto border">
        <table class="table-md md:table-lg table-zebra table">
          <thead>
            <tr>
              <th class="w-32 text-sm uppercase">{{ $t("cloud-search.col-time") }}</th>
              <th class="w-16 text-sm uppercase">{{ $t("cloud-search.col-level") }}</th>
              <th class="w-1 text-sm uppercase">{{ $t("cloud-search.col-container") }}</th>
              <th class="text-sm uppercase">{{ $t("cloud-search.col-message") }}</th>
            </tr>
          </thead>
          <tbody class="bg-base-300/30">
            <tr
              v-for="(hit, i) in hits"
              :key="i"
              class="hover:bg-base-100/80!"
              :class="{ 'cursor-pointer': isLive(hit), 'opacity-60': !isLive(hit) }"
              @click="isLive(hit) && openContainer(hit)"
            >
              <td class="text-base-content/50 font-mono text-xs whitespace-nowrap">{{ formatTs(hit.ts) }}</td>
              <td>
                <span :class="levelColor(hit.level)" class="font-mono text-xs font-semibold uppercase">
                  {{ hit.level || "info" }}
                </span>
              </td>
              <td class="whitespace-nowrap">
                <span class="inline-flex items-center gap-2">
                  <span :class="isLive(hit) ? 'text-base-content' : 'text-base-content/50'">
                    {{ hit.containerName }}
                  </span>
                  <span
                    v-if="!isLive(hit)"
                    :title="$t('cloud-search.container-removed')"
                    class="badge badge-sm badge-ghost"
                  >
                    {{ $t("cloud-search.container-removed-pill") }}
                  </span>
                </span>
              </td>
              <td>
                <span class="font-mono text-xs" v-html="highlight(hit.message, committedQuery)"></span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Cloud-not-available state -->
      <div
        v-if="!cloudSearch.available.value && committedQuery"
        class="bg-base-200 border-base-content/10 rounded-lg border p-6 text-center"
      >
        <mdi:cloud-off-outline class="text-base-content/30 mx-auto mb-2 size-8" />
        <p class="text-base-content/60 text-sm">
          {{
            cloudConfig?.linked ? $t("cloud-search.enable-streaming-to-search") : $t("cloud-search.connect-to-enable")
          }}
        </p>
        <RouterLink to="/settings/cloud" class="btn btn-primary btn-sm mt-3">
          {{ $t("settings.cloud") || "Cloud settings" }}
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

watch(
  () => route.query.q,
  (q) => {
    committedQuery.value = readQ(q);
  },
);

function formatTs(ns: number): string {
  const d = new Date(ns / 1e6);
  return d.toLocaleTimeString([], { hour12: false }) + "." + String(d.getMilliseconds()).padStart(3, "0");
}

function levelColor(level: string): string {
  switch ((level || "").toLowerCase()) {
    case "error":
    case "fatal":
      return "text-error";
    case "warn":
    case "warning":
      return "text-warning";
    case "info":
      return "text-info";
    default:
      return "text-base-content/50";
  }
}

// Safe with v-html: escapeHtml runs first, then <mark> tags are added against
// a regex anchored on the (already-escaped) needle. Don't drop the escape
// thinking it's redundant — the message comes from indexed log content.
function highlight(message: string, q: string): string {
  if (!q) return escapeHtml(message);
  const pattern = q.replace(/[-/\\^$*+?.()|[\]{}]/g, "\\$&");
  const re = new RegExp(`(${pattern})`, "gi");
  return escapeHtml(message).replace(re, '<mark class="bg-warning/30 text-warning rounded px-0.5">$1</mark>');
}

function escapeHtml(s: string): string {
  return s.replace(
    /[&<>"']/g,
    (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c] as string,
  );
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
