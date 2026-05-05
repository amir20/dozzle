<template>
  <div class="dropdown dropdown-open w-full shadow-md">
    <div class="input input-xl input-primary flex w-full items-center">
      <mdi:magnify class="flex size-8" />
      <input
        tabindex="0"
        class="input-ghost flex-1 px-1"
        ref="input"
        @keydown.down="selectedIndex = Math.min(selectedIndex + 1, data.length - 1)"
        @keydown.up="selectedIndex = Math.max(selectedIndex - 1, 0)"
        @keydown.enter.exact="onEnter"
        @keydown.shift.enter.exact.prevent="runLogSearch"
        @keydown.alt.enter="addColumn(data[selectedIndex].item)"
        v-model="query"
        :placeholder="$t('placeholder.search-containers')"
      />
      <form method="dialog" class="flex">
        <button v-if="isMobile">
          <mdi:close />
        </button>
        <button v-else class="swap hover:swap-active outline-hidden">
          <mdi:keyboard-esc class="swap-off" />
          <mdi:close class="swap-on" />
        </button>
      </form>
    </div>
    <div
      class="dropdown-content bg-base-100 relative! mt-2 max-h-[calc(100dvh-20rem)] w-full overflow-y-scroll rounded-md border-y-8 border-transparent px-2"
      tabindex="0"
      v-if="results.length || logSearchVisible"
    >
      <ul class="menu w-auto" v-if="results.length">
        <li v-for="(result, index) in data" ref="listItems">
          <a
            class="grid auto-cols-max grid-cols-[min-content_auto] gap-2 py-4"
            @click.prevent="selected(result.item)"
            :class="{ 'menu-focus': index === selectedIndex }"
          >
            <div :class="{ 'text-primary': result.item.state === 'running' }">
              <template v-if="result.item.type === 'container'">
                <octicon:container-24 />
              </template>
              <template v-else-if="result.item.type === 'service'">
                <ph:stack-simple />
              </template>
              <template v-else-if="result.item.type === 'stack'">
                <ph:stack />
              </template>
            </div>
            <div class="truncate">
              <template v-if="config.hosts.length > 1 && result.item.host">
                <span class="font-light">{{ result.item.host }}</span> /
              </template>
              <span data-name v-html="matchedName(result)"></span>
            </div>

            <RelativeTime :date="result.item.created" class="text-xs font-light" />
            <span
              @click.stop.prevent="addColumn(result.item)"
              :title="$t('tooltip.pin-column')"
              class="hover:text-secondary"
            >
              <ic:sharp-keyboard-return v-if="index === selectedIndex" />
              <cil:columns v-else-if="result.item.type === 'container'" />
            </span>
          </a>
        </li>
      </ul>

      <!-- Cloud log search CTA: appears when there's a query, regardless of container matches -->
      <div
        v-if="logSearchVisible"
        class="border-base-content/10 border-t"
        :class="{ 'cursor-pointer': cloudSearch.available.value, 'opacity-60': !cloudSearch.available.value }"
        @click="cloudSearch.available.value && runLogSearch()"
      >
        <div
          class="flex items-center gap-3 px-3 py-3"
          :class="cloudSearch.available.value ? 'bg-primary/5 hover:bg-primary/10' : 'bg-base-200/30'"
        >
          <mdi:cloud-search-outline
            class="size-5 shrink-0"
            :class="cloudSearch.available.value ? 'text-primary' : 'text-base-content/40'"
          />
          <div class="flex min-w-0 flex-1 flex-col">
            <span
              class="truncate text-sm font-semibold"
              :class="cloudSearch.available.value ? 'text-primary' : 'text-base-content/60'"
            >
              <i18n-t keypath="cloud-search.search-logs-for">
                <template #query>
                  <span class="font-mono">{{ query }}</span>
                </template>
              </i18n-t>
            </span>
            <span class="text-base-content/50 mt-0.5 flex items-center gap-1 text-xs">
              <template v-if="cloudSearch.available.value">
                <mdi:flash class="text-primary size-3" />
                {{ $t("cloud-search.across-containers") }}
              </template>
              <template v-else-if="cloudConfig?.linked && !cloudConfig.streamLogs">
                <mdi:cloud-off-outline class="size-3" />
                <RouterLink to="/settings/cloud" class="link link-hover" @click.stop>
                  {{ $t("cloud-search.enable-streaming-to-search") }}
                </RouterLink>
              </template>
              <template v-else>
                <mdi:cloud-off-outline class="size-3" />
                <RouterLink to="/settings/cloud" class="link link-hover" @click.stop>
                  {{ $t("cloud-search.connect-to-enable") }}
                </RouterLink>
              </template>
            </span>
          </div>
          <kbd class="kbd kbd-xs">⇧</kbd>
          <kbd class="kbd kbd-xs">↵</kbd>
        </div>
      </div>
    </div>

    <!-- Footer: cloud status + keyboard hints. Always visible while the modal
         is open so users know log search is available before they type. -->
    <div
      class="bg-base-200/40 border-base-content/5 text-base-content/50 mt-2 flex items-center gap-3 rounded-md border px-3 py-1.5 text-xs"
    >
      <span v-if="results.length" class="flex items-center gap-1">
        <kbd class="kbd kbd-xs">↵</kbd> {{ $t("cloud-search.open-container") }}
      </span>
      <span v-if="cloudSearch.available.value && logSearchVisible" class="flex items-center gap-1">
        <kbd class="kbd kbd-xs">⇧</kbd><kbd class="kbd kbd-xs">↵</kbd>
        {{ $t("cloud-search.search-logs-shortcut") }}
      </span>

      <!-- Cloud status — muted text with mint cloud icon to match the design.
           Stays in the same color band as the kbd hints so it doesn't fight
           the "Search logs for X" CTA above. -->
      <span v-if="cloudSearch.available.value" class="ml-auto flex items-center gap-1.5">
        <mdi:cloud-check-outline class="text-primary size-3.5" />
        {{ $t("cloud-search.cloud-connected") }}
      </span>
      <span v-else-if="cloudConfig?.linked" class="ml-auto flex items-center gap-1.5">
        <mdi:cloud-off-outline class="size-3.5" />
        <RouterLink to="/settings/cloud" class="link link-hover" @click.stop>
          {{ $t("cloud-search.enable-streaming-to-search") }}
        </RouterLink>
      </span>
      <span v-else class="ml-auto flex items-center gap-1.5">
        <mdi:cloud-off-outline class="size-3.5" />
        <RouterLink to="/settings/cloud" class="link link-hover" @click.stop>
          {{ $t("cloud-search.connect-to-enable") }}
        </RouterLink>
      </span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ContainerState } from "@/types/Container";
import { useFuse } from "@vueuse/integrations/useFuse";
import { type FuseResult } from "fuse.js";
import { useCloudConfig } from "@/composable/cloudConfig";
import { useCloudLogSearch } from "@/composable/cloudLogSearch";

const close = defineEmit();

const query = ref("");
const input = ref<HTMLInputElement>();
const listItems = ref<HTMLInputElement[]>();
const selectedIndex = ref(0);

const router = useRouter();
const containerStore = useContainerStore();
const pinnedStore = usePinnedLogsStore();
const { visibleContainers } = storeToRefs(containerStore);

const swarmStore = useSwarmStore();
const { stacks, services } = storeToRefs(swarmStore);

const { cloudConfig } = useCloudConfig();
// We don't render the live cloud search results inside the popup (the design
// keeps the popup lightweight) but we still mount the composable so the
// "Search logs for X" CTA can react to availability and so an in-flight
// query is warm by the time the user hits ⇧↵.
const cloudSearch = useCloudLogSearch(query);

const logSearchVisible = computed(() => query.value.trim().length > 0);

onMounted(async () => {
  const dialog = input.value?.closest("dialog");
  if (dialog) {
    const animations = dialog.getAnimations();
    await Promise.all(animations.map((animation) => animation.finished));
    input.value?.focus();
  }
});

type Item = {
  id: string;
  created: Date;
  name: string;
  state?: ContainerState;
  host?: string;
  type: "container" | "service" | "stack";
};

const list = computed(() => {
  const items: Item[] = [];

  for (const container of visibleContainers.value) {
    items.push({
      id: container.id,
      created: container.created,
      name: container.name,
      state: container.state,
      host: container.hostLabel,
      type: "container",
    });
  }

  for (const service of services.value) {
    items.push({
      id: service.name,
      created: service.updatedAt,
      name: service.name,
      state: "running",
      type: "service",
    });
  }

  for (const stack of stacks.value) {
    items.push({
      id: stack.name,
      created: stack.updatedAt,
      name: stack.name,
      state: "running",
      type: "stack",
    });
  }

  return items;
});

const { results: fuseResults } = useFuse(query, list, {
  fuseOptions: {
    keys: ["name", "host"],
    includeScore: true,
    useExtendedSearch: true,
    threshold: 0.3,
    includeMatches: true,
  },
});

const results = computed(() => (query.value ? fuseResults.value : []));

const data = computed(() => {
  return [...results.value].sort((a: FuseResult<Item>, b: FuseResult<Item>) => {
    if (a.score === b.score) {
      if (a.item.state === b.item.state) {
        return b.item.created.getTime() - a.item.created.getTime();
      } else if (a.item.state === "running" && b.item.state !== "running") {
        return -1;
      } else {
        return 1;
      }
    } else {
      return (a.score ?? 0) - (b.score ?? 0);
    }
  });
});

watch(query, (data) => {
  if (data.length > 0) {
    selectedIndex.value = 0;
  }
});

watch(selectedIndex, () => {
  listItems.value?.[selectedIndex.value].scrollIntoView({ block: "end" });
});

function selected(item: Item) {
  if (item.type === "container") {
    router.push({ name: "/container/[id]", params: { id: item.id } });
  } else if (item.type === "service") {
    router.push({ name: "/service/[name]", params: { name: item.id } });
  } else if (item.type === "stack") {
    router.push({ name: "/stack/[name]", params: { name: item.id } });
  }
  close();
}

function onEnter() {
  // Plain Enter prefers a container match if one is selected. With no
  // container matches (cloud-only query like "OOM"), fall back to log search
  // so the user isn't stuck on a popup that does nothing.
  if (data.value.length > 0) {
    selected(data.value[selectedIndex.value].item);
  } else if (cloudSearch.available.value && logSearchVisible.value) {
    runLogSearch();
  }
}

function runLogSearch() {
  if (!cloudSearch.available.value) return;
  const q = query.value.trim();
  if (!q) return;
  router.push({ path: "/cloud/search", query: { q } });
  close();
}

function addColumn(container: { id: string }) {
  pinnedStore.pinContainer(container);
  close();
}

function matchedName({ item, matches = [] }: FuseResult<Item>) {
  const matched = matches.find((match) => match.key === "name");
  if (matched) {
    const { indices } = matched;
    const result = [];
    let lastIndex = 0;
    for (const [start, end] of indices) {
      if (lastIndex > start) continue;
      result.push(item.name.slice(lastIndex, start));
      result.push(`<mark>${item.name.slice(start, end + 1)}</mark>`);
      lastIndex = end + 1;
    }
    result.push(item.name.slice(lastIndex));
    return result.join("");
  } else {
    return item.name;
  }
}
</script>

<style scoped>
@reference "@/main.css";
:deep(mark) {
  @apply bg-transparent text-inherit underline underline-offset-2;
}
</style>
