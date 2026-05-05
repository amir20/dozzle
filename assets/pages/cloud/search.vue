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
            <span>Searching…</span>
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
            <span class="font-mono">{{ hits.length }} hits</span>
          </template>
        </div>
      </div>

      <!-- Results list -->
      <div v-if="hits.length" class="bg-base-100 border-base-content/10 overflow-hidden rounded-lg border">
        <div
          v-for="(hit, i) in hits"
          :key="i"
          class="border-base-content/5 hover:bg-base-200/50 grid cursor-pointer grid-cols-[8rem_4rem_1fr] gap-3 border-b px-4 py-2 font-mono text-xs last:border-b-0"
          @click="openContainer(hit)"
        >
          <span class="text-base-content/40">{{ formatTs(hit.ts) }}</span>
          <span :class="levelColor(hit.level)" class="font-semibold uppercase">{{ hit.level || "info" }}</span>
          <div class="flex min-w-0 items-baseline gap-2">
            <span class="text-base-content/60 shrink-0">{{ hit.containerName }}</span>
            <span class="text-base-content truncate" v-html="highlight(hit.message, committedQuery)"></span>
          </div>
        </div>
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

const committedQuery = ref((route.query.q as string) || "");

const { cloudConfig } = useCloudConfig();
const cloudSearch = useCloudLogSearch(committedQuery);
const hits = computed<CloudLogHit[]>(() => cloudSearch.results.value);

watch(
  () => route.query.q,
  (q) => {
    committedQuery.value = (q as string) || "";
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
  router.push({
    name: "/container/[id]",
    params: { id: hit.containerId },
    query: { q: committedQuery.value, t: String(hit.ts) },
  });
}
</script>
