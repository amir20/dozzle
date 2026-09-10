<template>
  <!-- Single bordered card containing the input, results, and footer in one
       frame. No daisyUI input/dropdown chrome. -->
  <div class="bg-base-200 border-base-content/15 w-full overflow-hidden rounded-xl border shadow-2xl">
    <!-- Input row -->
    <div class="flex items-center gap-3 px-4 py-3.5">
      <mdi:magnify
        class="size-5 shrink-0"
        :class="cloudSearch.available.value ? 'text-primary' : 'text-base-content/60'"
      />
      <input
        tabindex="0"
        class="text-base-content placeholder:text-base-content/40 min-w-0 flex-1 bg-transparent text-base outline-none"
        ref="input"
        @keydown.down.prevent="move(1)"
        @keydown.up.prevent="move(-1)"
        @keydown.enter.exact.prevent="onEnter"
        @keydown.shift.enter.exact.prevent="runLogSearch"
        @keydown.alt.enter.exact.prevent="onPin"
        v-model="query"
        :placeholder="placeholderCopy"
      />
      <button
        v-if="query"
        type="button"
        class="text-base-content/40 hover:text-base-content shrink-0 cursor-pointer"
        :title="$t('toolbar.clear')"
        @click="clearQuery"
      >
        <mdi:close-circle class="size-4" />
      </button>
      <form method="dialog" class="flex shrink-0">
        <button v-if="isMobile" class="icon-btn text-base-content/50 hover:text-base-content">
          <mdi:close class="size-5" />
        </button>
        <button v-else class="cursor-pointer">
          <kbd class="kbd kbd-xs">esc</kbd>
        </button>
      </form>
    </div>

    <!-- Body. Always has something in it: on an empty query it offers the
         container actions for the current page plus the most recent
         containers, so the palette is never a dead box. -->
    <div class="border-base-content/10 border-t">
      <div class="max-h-[55vh] overflow-y-auto overscroll-contain pb-1.5">
        <!-- Nothing matched locally. The cloud row below still gives the query
             somewhere to go, so this is a note and not a dead end. -->
        <div v-if="noLocalMatches" class="text-base-content/40 px-4 pt-3 pb-1 text-sm" data-testid="no-matches">
          {{ $t("cloud-search.no-matches") }}
        </div>

        <template v-for="group in groups" :key="group.id">
          <div
            class="bg-base-200 text-base-content/40 sticky top-0 z-10 flex items-center gap-1.5 px-4 pt-3 pb-1.5 text-[11px] font-semibold tracking-wider uppercase"
          >
            {{ group.label }}
            <span v-if="group.entries.length > 1" class="text-base-content/25">{{ group.entries.length }}</span>
          </div>
          <ul class="px-1.5">
            <li v-for="entry in group.entries" :key="entry.key" :ref="(el) => setItemRef(el, entry.index)">
              <a
                class="flex cursor-pointer items-center gap-3 rounded-lg px-2.5 py-2"
                :class="entry.index === selectedIndex ? 'bg-base-content/10' : 'hover:bg-base-content/5'"
                @mousemove="selectedIndex = entry.index"
                @click.prevent="activate(entry)"
              >
                <!-- Command -->
                <template v-if="entry.kind === 'command'">
                  <component :is="entry.command.icon" class="text-base-content/60 size-4 shrink-0" />
                  <span class="min-w-0 flex-1 truncate text-sm">{{ entry.command.title }}</span>
                </template>

                <!-- Container, service or stack -->
                <template v-else-if="entry.kind === 'container'">
                  <ContainerIcon
                    v-if="entry.item.type === 'container' && entry.item.icon"
                    :state="entry.item.state ?? 'running'"
                    :slug="entry.item.icon"
                    class="size-5 shrink-0"
                  />
                  <div
                    v-else
                    class="shrink-0"
                    :class="entry.item.state === 'running' ? 'text-primary' : 'text-base-content/50'"
                  >
                    <template v-if="entry.item.type === 'container'">
                      <octicon:container-24 class="size-4" />
                    </template>
                    <template v-else-if="entry.item.type === 'service'">
                      <ph:stack-simple class="size-4" />
                    </template>
                    <template v-else-if="entry.item.type === 'stack'">
                      <ph:stack class="size-4" />
                    </template>
                  </div>
                  <div class="flex min-w-0 flex-1 items-center gap-2">
                    <div class="min-w-0 truncate text-sm">
                      <template v-if="config.hosts.length > 1 && entry.item.host">
                        <span class="text-base-content/50 font-light">{{ entry.item.host }}</span>
                        <span class="text-base-content/30"> / </span>
                      </template>
                      <span class="text-base-content" data-name v-html="matchedName(entry.result)"></span>
                    </div>
                    <span
                      v-if="entry.item.type !== 'container'"
                      class="border-base-content/15 text-base-content/50 shrink-0 rounded border px-1.5 py-px text-[10px] uppercase"
                    >
                      {{ entry.item.type }}
                    </span>
                    <span
                      v-else-if="entry.item.state && entry.item.state !== 'running'"
                      class="border-base-content/15 text-base-content/50 shrink-0 rounded border px-1.5 py-px text-[10px]"
                    >
                      {{ entry.item.state }}
                    </span>
                  </div>
                  <RelativeTime :date="entry.item.created" class="text-base-content/40 shrink-0 text-xs" />
                  <button
                    v-if="entry.item.type === 'container' && !isMobile"
                    type="button"
                    class="text-base-content/40 hover:text-secondary shrink-0 cursor-pointer"
                    :title="$t('tooltip.pin-column')"
                    @click.stop.prevent="addColumn(entry.item)"
                  >
                    <cil:columns class="size-4" />
                  </button>
                </template>

                <!-- Ask the assistant -->
                <template v-else-if="entry.kind === 'ask'">
                  <mdi:message-outline class="text-primary size-5 shrink-0" />
                  <div class="flex min-w-0 flex-1 flex-col">
                    <span class="text-primary truncate text-sm">
                      <i18n-t keypath="cloud-chat.ask-for">
                        <template #query>
                          <span class="font-mono">{{ query }}</span>
                        </template>
                      </i18n-t>
                    </span>
                    <span class="text-base-content/50 mt-0.5 truncate text-xs">
                      {{ $t("cloud-chat.ask-hint") }}
                    </span>
                  </div>
                </template>

                <!-- Cloud log search -->
                <template v-else>
                  <mdi:cloud-search-outline
                    class="size-5 shrink-0"
                    :class="cloudSearch.available.value ? 'text-primary' : 'text-base-content/40'"
                  />
                  <div class="flex min-w-0 flex-1 flex-col">
                    <span class="truncate text-sm" :class="cloudSearch.available.value ? 'text-primary' : ''">
                      <i18n-t keypath="cloud-search.search-logs-for">
                        <template #query>
                          <span class="font-mono">{{ query }}</span>
                        </template>
                      </i18n-t>
                    </span>
                    <span class="text-base-content/50 mt-0.5 truncate text-xs">
                      <template v-if="cloudSearch.available.value">
                        {{ $t("cloud-search.across-containers") }}
                      </template>
                      <template v-else-if="cloudConfig?.linked">
                        {{ $t("cloud-search.enable-streaming-to-search") }}
                      </template>
                      <template v-else>
                        {{ $t("cloud-search.connect-to-enable") }}
                      </template>
                    </span>
                  </div>
                  <template v-if="cloudSearch.available.value">
                    <kbd class="kbd kbd-xs shrink-0">⇧</kbd>
                    <kbd class="kbd kbd-xs shrink-0">↵</kbd>
                  </template>
                </template>
              </a>
            </li>
          </ul>
        </template>
      </div>
    </div>

    <!-- Footer: what the keys do for the current selection, plus cloud status. -->
    <div
      class="bg-base-300/40 border-base-content/10 text-base-content/50 flex items-center gap-3 border-t px-4 py-2 text-[11px]"
    >
      <span v-if="flatEntries.length > 1" class="hidden items-center gap-1 sm:flex">
        <kbd class="kbd kbd-xs">↑</kbd><kbd class="kbd kbd-xs">↓</kbd>
        <span class="ml-0.5">{{ $t("cloud-search.nav-hint") }}</span>
      </span>
      <span v-if="enterHint" class="hidden items-center gap-1 sm:flex">
        <kbd class="kbd kbd-xs">↵</kbd> <span class="ml-0.5">{{ enterHint }}</span>
      </span>
      <span v-if="selectedEntry?.kind === 'container'" class="hidden items-center gap-1 sm:flex">
        <kbd class="kbd kbd-xs">⌥</kbd><kbd class="kbd kbd-xs">↵</kbd>
        <span class="ml-0.5">{{ $t("cloud-search.pin-shortcut") }}</span>
      </span>

      <!-- Cloud status. Skipped while the log search row is on screen, which
           already carries the same call to action. -->
      <span v-if="cloudSearch.available.value" class="ml-auto flex shrink-0 items-center gap-1.5">
        <mdi:cloud-check-outline class="text-primary size-3.5" />
        {{ $t("cloud-search.cloud-connected") }}
      </span>
      <template v-else-if="logSearchVisible"></template>
      <span v-else-if="cloudConfig?.linked" class="ml-auto flex shrink-0 items-center gap-1.5">
        <mdi:cloud-off-outline class="size-3.5" />
        <RouterLink to="/settings/cloud" class="link link-hover" @click.stop>
          {{ $t("cloud-search.enable-streaming-to-search") }}
        </RouterLink>
      </span>
      <span v-else class="ml-auto flex shrink-0 items-center gap-1.5">
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
import { useCloudConfig } from "@/composable/cloud/cloudConfig";
import { useCloudLogSearch } from "@/composable/cloud/cloudLogSearch";
import { useCommands, type Command } from "@/composable/app/commands";

const close = defineEmit();

const { ask, openPane } = useCloudChat();
const { linked: cloudLinked } = useCloudSurface();
const view = useViewContext();

const router = useRouter();
const route = useRoute();

// How many containers to offer before the user has typed anything.
const RECENT_LIMIT = 6;

// Prefill with the current /cloud/search query so the user can refine
// without retyping. Empty everywhere else. Null-safe for unit tests
// that mount the component without a router context.
const initialQuery = route?.path === "/cloud/search" && typeof route.query?.q === "string" ? route.query.q : "";
const query = ref(initialQuery);
const input = ref<HTMLInputElement>();
const listItems = ref<(Element | null)[]>([]);
const selectedIndex = ref(0);

// Function ref into a single flat array so every group shares one selection
// index for arrow-key navigation and scroll-into-view.
function setItemRef(el: any, index: number) {
  listItems.value[index] = (el?.$el ?? el) as Element | null;
}

const containerStore = useContainerStore();
const pinnedStore = usePinnedLogsStore();
const { visibleContainers } = storeToRefs(containerStore);

const swarmStore = useSwarmStore();
const { stacks, services } = storeToRefs(swarmStore);

const { cloudConfig } = useCloudConfig();
// Mounted only so the log search row can read `available`. We don't render
// the hits inside the popup. The composable's debounced watch short-circuits
// on an empty query, so opening the modal alone does not fire a request.
const cloudSearch = useCloudLogSearch(query);

const trimmedQuery = computed(() => query.value.trim());
const logSearchVisible = computed(() => trimmedQuery.value.length > 0);

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
  icon?: string;
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
      icon: container.icon,
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
  trimmedQuery.value ? commandFuseResults.value.map((r) => r.item) : contextCommands.value,
);

const containerResults = computed<FuseResult<Item>[]>(() => {
  if (!trimmedQuery.value) return [];
  return [...fuseResults.value].sort((a, b) => {
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

// Empty-query suggestions: running containers first, newest first. Shaped like
// a Fuse result so the rows render through the same branch.
const recentResults = computed<FuseResult<Item>[]>(() => {
  if (trimmedQuery.value) return [];
  return [...list.value]
    .sort((a, b) => {
      if (a.state === b.state) return b.created.getTime() - a.created.getTime();
      return a.state === "running" ? -1 : 1;
    })
    .slice(0, RECENT_LIMIT)
    .map((item, refIndex) => ({ item, refIndex }));
});

type EntryKind =
  | { kind: "command"; command: Command }
  | { kind: "container"; result: FuseResult<Item>; item: Item }
  | { kind: "logs" }
  | { kind: "ask" };
type PendingEntry = EntryKind & { key: string };
type Entry = EntryKind & { key: string; index: number };
type Group = { id: string; label: string; entries: Entry[] };

// One flat, ordered list of everything selectable, sliced into labeled groups.
// Each row carries its position in `flatEntries`, so arrow keys, Enter and
// scroll-into-view all agree without any per-section offset arithmetic.
const groups = computed<Group[]>(() => {
  const result: Group[] = [];
  let index = 0;

  const push = (id: string, label: string, entries: PendingEntry[]) => {
    if (!entries.length) return;
    result.push({ id, label, entries: entries.map((entry) => ({ ...entry, index: index++ })) as Entry[] });
  };

  // Containers lead: this is a log viewer, so a typed query is far more often a
  // container name than a command. Enter on the first row opens a container.
  push(
    "containers",
    trimmedQuery.value ? t("cloud-search.containers-section") : t("command-palette.section-recent"),
    (trimmedQuery.value ? containerResults.value : recentResults.value).map((result) => ({
      kind: "container",
      result,
      item: result.item,
      key: `${result.item.type}:${result.item.id}`,
    })),
  );

  push(
    "commands",
    t("command-palette.section-commands"),
    commandEntries.value.map((command) => ({ kind: "command", command, key: `command:${command.id}` })),
  );

  if (logSearchVisible.value) {
    push("logs", t("cloud-search.section-logs"), [{ kind: "logs", key: "logs" }]);
  }

  // Last row, and only with something typed: the palette is for finding
  // things, and asking is what you do when finding did not answer it.
  if (askVisible.value) {
    push("ask", t("cloud-chat.section-ask"), [{ kind: "ask", key: "ask" }]);
  }

  return result;
});

const flatEntries = computed(() => groups.value.flatMap((group) => group.entries));
const selectedEntry = computed<Entry | undefined>(() => flatEntries.value[selectedIndex.value]);
// The assistant answers about the logs on screen, so the row is offered only
// where there are logs on screen. Elsewhere it would open a panel the layout
// does not mount.
const { available: railAvailable } = useCloudRail();
const askVisible = computed(() => cloudLinked.value && railAvailable.value && !!trimmedQuery.value);
const noLocalMatches = computed(
  () => !!trimmedQuery.value && !commandEntries.value.length && !containerResults.value.length,
);

const enterHint = computed(() => {
  switch (selectedEntry.value?.kind) {
    case "command":
      return t("command-palette.run-hint");
    case "container":
      return t("cloud-search.open-container");
    case "logs":
      return t("cloud-search.search-logs-shortcut");
    case "ask":
      return t("cloud-chat.ask-shortcut");
    default:
      return "";
  }
});

// Reset to the top only when the user types. Live SSE container add/remove
// changes the list too, and resetting on that would snap the selection back
// to 0 while the palette is open.
watch(trimmedQuery, () => {
  selectedIndex.value = 0;
});

// Keep the selection in bounds when the result count shrinks underneath it.
watch(
  () => flatEntries.value.length,
  (count) => {
    if (selectedIndex.value > count - 1) {
      selectedIndex.value = Math.max(count - 1, 0);
    }
  },
);

watch(selectedIndex, () => {
  listItems.value?.[selectedIndex.value]?.scrollIntoView({ block: "nearest" });
});

// Arrow keys wrap around, which keeps the log search row one keystroke away
// from the top of a long container list.
function move(delta: number) {
  const count = flatEntries.value.length;
  if (!count) return;
  selectedIndex.value = (selectedIndex.value + delta + count) % count;
}

function clearQuery() {
  query.value = "";
  input.value?.focus();
}

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

function activate(entry: Entry) {
  if (entry.kind === "command") {
    runCommand(entry.command);
  } else if (entry.kind === "container") {
    selected(entry.item);
  } else if (entry.kind === "ask") {
    askAssistant();
  } else {
    runLogSearch();
  }
}

function onEnter() {
  const entry = selectedEntry.value;
  if (entry) activate(entry);
}

function onPin() {
  // Alt+Enter pins a container column. Only meaningful when a container row is
  // selected, not a command or the log search row.
  const entry = selectedEntry.value;
  if (entry?.kind === "container" && entry.item.type === "container") addColumn(entry.item);
}

// The palette is already a text box the user typed a question into, so this
// goes straight through rather than opening an empty pane to retype it in.
function askAssistant() {
  const q = trimmedQuery.value;
  if (!q) return;
  openPane();
  ask(q, view.value);
  close();
}

function runLogSearch() {
  const q = trimmedQuery.value;
  if (!q) return;
  // Not linked (or not streaming) yet: send the user where they can turn it on
  // rather than silently doing nothing.
  if (!cloudSearch.available.value) {
    router.push("/settings/cloud");
    close();
    return;
  }
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
