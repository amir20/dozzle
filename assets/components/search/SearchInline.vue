<template>
  <!-- The topbar search entry, and where the assistant lives in the same bar.
       This is container search first: cloud log search is an extra it picks up
       when the instance is linked, which is why it must not be gated on the cloud
       role. Without cloud it falls back to the plain container label. -->
  <div
    class="bg-base-200 border-base-content/15 hover:border-base-content/30 flex h-9 w-full items-stretch rounded-md border transition-colors"
  >
    <button
      type="button"
      data-testid="search"
      class="hover:bg-base-300/60 flex min-w-0 flex-1 items-center gap-2 rounded-l-md px-3 text-left transition-colors"
      :class="{ 'rounded-r-md': !railAvailable }"
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
      <span v-if="canHover" class="ml-auto flex items-center gap-1">
        <kbd class="kbd kbd-xs">⌘</kbd>
        <kbd class="kbd kbd-xs">K</kbd>
      </span>
    </button>
    <!-- The same chat the rail opens, one click from the bar every page has. It
         sits beside search rather than inside the palette because asking is the
         thing people did not know they could do here. -->
    <button
      v-if="railAvailable"
      type="button"
      data-testid="search-ask"
      class="border-base-content/15 hover:bg-base-300/60 flex shrink-0 items-center gap-1.5 rounded-r-md border-l px-3 text-sm transition-colors"
      :class="chatOpen ? 'bg-base-300/60 text-base-content' : 'text-base-content/80'"
      :aria-pressed="chatOpen"
      :title="`${$t('cloud-rail.chat')} (⇧${mod}K)`"
      @click="toggleRail('chat')"
    >
      <mdi:message-outline class="text-primary size-4" />
      <span class="whitespace-nowrap">{{ $t("cloud-rail.chat") }}</span>
    </button>
  </div>
</template>

<script lang="ts" setup>
import { useFuzzySearch } from "@/composable/app/fuzzySearch";
import { useCloudConfig } from "@/composable/cloud/cloudConfig";
import { modKey } from "@/composable/app/commands";

// The chord's modifier as this platform names it: ⌘ on a Mac, Ctrl elsewhere.
const mod = modKey();

const { openSearch } = useFuzzySearch();
const { cloudConfig } = useCloudConfig();
const cloudReady = computed(() => !!cloudConfig.value?.linked && !!cloudConfig.value?.streamLogs);
const { available: railAvailable, panel, toggleRail } = useCloudRail();
const chatOpen = computed(() => panel.value === "chat");

const route = useRoute();
const activeQuery = computed(() =>
  route?.path === "/cloud/search" && typeof route.query?.q === "string" ? route.query.q : "",
);
</script>
