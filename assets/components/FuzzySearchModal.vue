<template>
  <!-- Single bordered card containing the input, results, and footer in one
       frame to match the design mock. No daisyUI input/dropdown chrome. -->
  <div class="bg-base-200 border-base-content/15 w-full overflow-hidden rounded-xl border shadow-2xl">
    <!-- Input row -->
    <div class="flex items-center gap-3 px-4 py-3.5">
      <mdi:magnify class="text-base-content/60 size-5 shrink-0" />
      <input
        tabindex="0"
        class="text-base-content placeholder:text-base-content/40 flex-1 bg-transparent text-base outline-none"
        ref="input"
        @keydown.down="selectedIndex = Math.min(selectedIndex + 1, totalCount - 1)"
        @keydown.up="selectedIndex = Math.max(selectedIndex - 1, 0)"
        @keydown.enter.exact="onEnter"
        @keydown.shift.enter.exact.prevent="runLogSearch"
        @keydown.alt.enter.exact.prevent="onPin"
        v-model="query"
        :placeholder="placeholderCopy"
      />
      <form method="dialog" class="flex">
        <button v-if="isMobile" class="text-base-content/50 hover:text-base-content">
          <mdi:close class="size-5" />
        </button>
        <button v-else>
          <kbd class="kbd kbd-xs">esc</kbd>
        </button>
      </form>
    </div>

    <!-- Body: results + log search CTA. Only renders when there is something
         to show — keeps the empty modal compact. -->
    <div v-if="totalCount || logSearchVisible" class="border-base-content/10 border-t">
      <!-- Commands section -->
      <template v-if="commandEntries.length">
        <div class="text-base-content/40 px-4 pt-3 pb-1.5 text-xs font-semibold tracking-wider uppercase">
          {{ $t("command-palette.section-commands") }} · {{ commandEntries.length }}
        </div>
        <ul class="pb-1">
          <li v-for="(command, index) in commandEntries" :ref="(el) => setItemRef(el, index)">
            <a
              class="hover:bg-base-content/5 flex cursor-pointer items-center gap-3 px-4 py-2"
              :class="{ 'bg-base-content/10': index === selectedIndex }"
              @click.prevent="runCommand(command)"
            >
              <component :is="command.icon" class="text-base-content/60 size-4 shrink-0" />
              <span class="min-w-0 flex-1 truncate text-sm">{{ command.title }}</span>
              <ic:sharp-keyboard-return v-if="index === selectedIndex" class="text-base-content/40 size-4" />
            </a>
          </li>
        </ul>
      </template>

      <!-- Containers section -->
      <template v-if="containerEntries.length">
        <div
          class="text-base-content/40 px-4 pt-3 pb-1.5 text-xs font-semibold tracking-wider uppercase"
          :class="{ 'border-base-content/10 mt-1 border-t': commandEntries.length }"
        >
          {{ $t("cloud-search.containers-section") }} · {{ containerEntries.length }}
        </div>
        <ul class="pb-1">
          <li v-for="(result, index) in containerEntries" :ref="(el) => setItemRef(el, commandEntries.length + index)">
            <a
              class="hover:bg-base-content/5 flex cursor-pointer items-center gap-3 px-4 py-2"
              :class="{ 'bg-base-content/10': commandEntries.length + index === selectedIndex }"
              @click.prevent="selected(result.item)"
            >
              <div :class="result.item.state === 'running' ? 'text-primary' : 'text-base-content/50'">
                <template v-if="result.item.type === 'container'">
                  <octicon:container-24 class="size-4" />
                </template>
                <template v-else-if="result.item.type === 'service'">
                  <ph:stack-simple class="size-4" />
                </template>
                <template v-else-if="result.item.type === 'stack'">
                  <ph:stack class="size-4" />
                </template>
              </div>
              <div class="min-w-0 flex-1 truncate text-sm">
                <template v-if="config.hosts.length > 1 && result.item.host">
                  <span class="text-base-content/50 font-light">{{ result.item.host }}</span>
                  <span class="text-base-content/30"> / </span>
                </template>
                <span class="text-base-content" data-name v-html="matchedName(result)"></span>
              </div>
              <RelativeTime :date="result.item.created" class="text-base-content/40 text-xs" />
              <span
                @click.stop.prevent="addColumn(result.item)"
                :title="$t('tooltip.pin-column')"
                class="text-base-content/40 hover:text-secondary"
              >
                <ic:sharp-keyboard-return v-if="commandEntries.length + index === selectedIndex" class="size-4" />
                <cil:columns v-else-if="result.item.type === 'container'" class="size-4" />
              </span>
            </a>
          </li>
        </ul>
      </template>

      <!-- Log search CTA -->
      <div
        v-if="logSearchVisible"
        class="border-base-content/10 border-t"
        :class="{ 'cursor-pointer': cloudSearch.available.value, 'opacity-70': !cloudSearch.available.value }"
        @click="cloudSearch.available.value && runLogSearch()"
      >
        <div
          class="flex items-center gap-3 px-4 py-3"
          :class="cloudSearch.available.value ? 'bg-primary/[0.07] hover:bg-primary/10' : ''"
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

    <!-- Footer: kbd hints + cloud status. Always present while the modal is
         open so users know log search is available before they type. -->
    <div
      class="bg-base-300/40 border-base-content/10 text-base-content/50 flex items-center gap-4 border-t px-4 py-2 text-xs"
    >
      <span v-if="totalCount" class="flex items-center gap-1.5">
        <kbd class="kbd kbd-xs">↵</kbd> {{ $t("cloud-search.open-container") }}
      </span>
      <span v-if="cloudSearch.available.value && logSearchVisible" class="flex items-center gap-1">
        <kbd class="kbd kbd-xs">⇧</kbd><kbd class="kbd kbd-xs">↵</kbd>
        <span class="ml-0.5">{{ $t("cloud-search.search-logs-shortcut") }}</span>
      </span>

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
import { useCommands, type Command } from "@/composable/commands";

const close = defineEmit();

const router = useRouter();
const route = useRoute();

// Prefill with the current /cloud/search query so the user can refine
// without retyping. Empty everywhere else. Null-safe for unit tests
// that mount the component without a router context.
const initialQuery = route?.path === "/cloud/search" && typeof route.query?.q === "string" ? route.query.q : "";
const query = ref(initialQuery);
const input = ref<HTMLInputElement>();
const listItems = ref<(Element | null)[]>([]);
const selectedIndex = ref(0);

// Function ref into a single flat array so Commands and Containers share one
// selection index for arrow-key navigation and scroll-into-view.
function setItemRef(el: any, index: number) {
  listItems.value[index] = (el?.$el ?? el) as Element | null;
}

const containerStore = useContainerStore();
const pinnedStore = usePinnedLogsStore();
const { visibleContainers } = storeToRefs(containerStore);

const swarmStore = useSwarmStore();
const { stacks, services } = storeToRefs(swarmStore);

const { cloudConfig } = useCloudConfig();
// Mounted only so the "Search logs for X" CTA can read `available`. We
// don't render the hits inside the popup. The composable's debounced
// watch short-circuits on empty query, so opening the modal alone does
// not fire a request.
const cloudSearch = useCloudLogSearch(query);

const logSearchVisible = computed(() => query.value.trim().length > 0);

const { t } = useI18n();
const placeholderCopy = computed(() =>
  cloudSearch.available.value ? t("cloud-search.modal-placeholder-cloud") : t("cloud-search.modal-placeholder-plain"),
);

onMounted(async () => {
  const dialog = input.value?.closest("dialog");
  if (dialog) {
    const animations = dialog.getAnimations();
    await Promise.all(animations.map((animation) => animation.finished));
    input.value?.focus();
    if (initialQuery) input.value?.select();
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

// Commands palette. Fuzzy-matched against title/keywords while typing; the
// context commands (container actions) show up front on an empty query.
const { commands, contextCommands } = useCommands();
const { results: commandFuseResults } = useFuse(query, commands, {
  fuseOptions: {
    keys: ["title", "keywords"],
    useExtendedSearch: true,
    threshold: 0.3,
  },
});
const commandEntries = computed<Command[]>(() =>
  query.value ? commandFuseResults.value.map((r) => r.item) : contextCommands.value,
);

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

// Container hits, mirrors the previously named `data` list for the template.
const containerEntries = computed(() => data.value);
const totalCount = computed(() => commandEntries.value.length + containerEntries.value.length);

// Reset to the top only when the user types. Live SSE container add/remove
// changes totalCount too, and resetting on that would snap the selection back
// to 0 while the palette is open.
watch(query, () => {
  selectedIndex.value = 0;
});

// Keep the selection in bounds when the result count shrinks underneath it.
watch(totalCount, (count) => {
  if (selectedIndex.value > count - 1) {
    selectedIndex.value = Math.max(count - 1, 0);
  }
});

watch(selectedIndex, () => {
  listItems.value?.[selectedIndex.value]?.scrollIntoView({ block: "nearest" });
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

async function runCommand(command: Command) {
  close();
  await command.perform();
}

function onEnter() {
  // Commands come first in the flat list, then containers. With nothing
  // selectable (cloud-only query like "OOM"), fall back to log search so the
  // user isn't stuck on a popup that does nothing.
  const commandCount = commandEntries.value.length;
  if (selectedIndex.value < commandCount) {
    runCommand(commandEntries.value[selectedIndex.value]);
  } else if (containerEntries.value.length > 0) {
    selected(containerEntries.value[selectedIndex.value - commandCount].item);
  } else if (cloudSearch.available.value && logSearchVisible.value) {
    runLogSearch();
  }
}

function onPin() {
  // Alt+Enter pins a container column. Only meaningful when a container row is
  // selected, not a command.
  const commandCount = commandEntries.value.length;
  if (selectedIndex.value >= commandCount) {
    const entry = containerEntries.value[selectedIndex.value - commandCount];
    if (entry?.item.type === "container") addColumn(entry.item);
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
