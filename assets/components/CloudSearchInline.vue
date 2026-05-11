<template>
  <button
    type="button"
    data-testid="search"
    class="bg-base-200 border-base-content/15 hover:border-primary/50 hover:bg-base-200/80 flex h-9 w-full items-center gap-2 rounded-md border px-3 text-left transition-colors"
    @click="openSearch"
  >
    <mdi:magnify class="size-4 shrink-0" :class="cloudReady ? 'text-primary' : 'text-base-content/60'" />
    <!-- Show the active query when we're on the cloud search page so the
         topbar reflects what the user is looking at. -->
    <span v-if="activeQuery" class="text-base-content truncate font-mono text-sm">{{ activeQuery }}</span>
    <span v-else class="text-base-content/60 truncate text-sm">
      <template v-if="cloudReady">{{ $t("cloud-search.hero-title-cloud") }}</template>
      <template v-else>{{ $t("cloud-search.hero-title-plain") }}</template>
    </span>
    <span class="ml-auto flex items-center gap-1">
      <kbd class="kbd kbd-xs">⌘</kbd>
      <kbd class="kbd kbd-xs">K</kbd>
    </span>
  </button>
</template>

<script lang="ts" setup>
import { useFuzzySearch } from "@/composable/fuzzySearch";
import { useCloudConfig } from "@/composable/cloudConfig";

const { openSearch } = useFuzzySearch();
const { cloudConfig } = useCloudConfig();
const cloudReady = computed(() => !!cloudConfig.value?.linked && !!cloudConfig.value?.streamLogs);

const route = useRoute();
const activeQuery = computed(() =>
  route?.path === "/cloud/search" && typeof route.query?.q === "string" ? route.query.q : "",
);
</script>
