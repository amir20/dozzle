<template>
  <!-- Single bordered card containing the input, results, and footer in one
       frame. No daisyUI input/dropdown chrome. -->
  <div class="bg-base-200 border-base-content/15 w-full overflow-hidden rounded-xl border shadow-2xl">
    <!-- Input row -->
    <div class="flex items-center gap-3 px-4 py-3.5">
      <mdi:magnify class="size-5 shrink-0" :class="cloudSearchAvailable ? 'text-primary' : 'text-base-content/60'" />
      <!-- Inside a container's actions: the container, as a crumb that goes back. -->
      <button
        v-if="drilled"
        type="button"
        class="border-base-content/15 text-base-content/70 hover:bg-base-content/5 flex max-w-[40%] shrink-0 items-center gap-1 rounded-md border px-2 py-0.5 text-sm transition-colors"
        :title="$t('command-palette.back-hint')"
        data-testid="drill-crumb"
        @click="drillOut"
      >
        <mdi:chevron-left class="size-4 shrink-0" />
        <span class="truncate">{{ drilled.name }}</span>
      </button>
      <input
        tabindex="0"
        class="text-base-content placeholder:text-base-content/40 min-w-0 flex-1 bg-transparent text-base outline-none"
        ref="input"
        @keydown.down.prevent="move(1)"
        @keydown.up.prevent="move(-1)"
        @keydown.enter.exact.prevent="onEnter"
        @keydown.shift.enter.exact.prevent="runLogSearch"
        @keydown.alt.enter.exact.prevent="onPin"
        @keydown.meta.enter.exact.prevent="onMerge"
        @keydown.ctrl.enter.exact.prevent="onMerge"
        @keydown.tab.exact="onTab"
        @keydown.right.exact="onRight"
        @keydown.left.exact="onBack"
        @keydown.delete.exact="onBack"
        @keydown.esc="onEscape"
        v-model="query"
        :placeholder="drilled ? $t('command-palette.actions-placeholder') : placeholderCopy"
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
            <span v-if="(group.count ?? group.entries.length) > 1" class="text-base-content/25">{{
              group.count ?? group.entries.length
            }}</span>
          </div>
          <ul class="px-1.5">
            <li v-for="entry in group.entries" :key="entry.key" :ref="(el) => setItemRef(el, entry.index)">
              <a
                class="flex cursor-pointer items-center gap-3 rounded-lg px-2.5 py-2"
                :class="entry.index === selectedIndex ? 'bg-base-content/10' : 'hover:bg-base-content/5'"
                @mousemove="selectedIndex = entry.index"
                @click.prevent="activate(entry, true)"
              >
                <!-- Command -->
                <template v-if="entry.kind === 'command'">
                  <component :is="entry.command.icon" class="text-base-content/60 size-4 shrink-0" />
                  <span v-if="isArmed(entry)" class="text-warning min-w-0 flex-1 truncate text-sm" data-testid="armed">
                    {{ entry.command.confirm }}
                  </span>
                  <template v-else>
                    <span class="min-w-0 flex-1 truncate text-sm">{{ entry.command.title }}</span>
                    <span v-if="entry.command.detail" class="text-base-content/40 shrink-0 font-mono text-xs">
                      {{ entry.command.detail }}
                    </span>
                    <span
                      v-if="entry.command.state !== undefined"
                      class="status-pill shrink-0"
                      :class="entry.command.state ? 'status-pill-primary' : 'status-pill-neutral'"
                      data-testid="state"
                    >
                      {{ entry.command.state ? $t("command-palette.state-on") : $t("command-palette.state-off") }}
                    </span>
                    <mdi:check v-if="entry.command.active" class="text-primary size-4 shrink-0" data-testid="active" />
                    <span v-if="entry.command.shortcut && canHover" class="hidden shrink-0 gap-0.5 sm:flex">
                      <kbd v-for="key in entry.command.shortcut" :key="key" class="kbd kbd-xs">{{ key }}</kbd>
                    </span>
                  </template>
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
                    <template v-else-if="entry.item.type === 'host'">
                      <mdi:server class="size-4" />
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
                      v-if="hasWaitingUpdate(entry.item)"
                      class="text-info size-2 shrink-0 rounded-full border-[1.5px] border-current"
                      :title="$t('toolbar.update-available')"
                      data-testid="update-dot"
                    ></span>
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
                  <RelativeTime
                    v-if="entry.item.type !== 'host'"
                    :date="entry.item.created"
                    class="text-base-content/40 shrink-0 text-xs"
                  />
                  <!-- Only offered where it would work: something is streaming. The
                       sidebar popup explains the rest. -->
                  <button
                    v-if="entry.item.type === 'container' && canToggle(entry.item.id)"
                    type="button"
                    class="text-base-content/40 hover:text-secondary relative shrink-0 after:absolute after:-inset-2 after:content-['']"
                    :class="{ 'text-secondary': isStreaming(entry.item.id) }"
                    :title="toggleTitle(entry.item.id)"
                    :aria-label="toggleTitle(entry.item.id)"
                    :aria-pressed="isStreaming(entry.item.id)"
                    @click.stop.prevent="mergeItem(entry.item)"
                  >
                    <ph:arrows-merge class="size-4" />
                  </button>
                  <button
                    v-if="entry.item.type === 'container' && !isMobile"
                    type="button"
                    class="text-base-content/40 hover:text-secondary relative shrink-0 after:absolute after:-inset-2 after:content-['']"
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
                      <i18n-t scope="global" keypath="cloud-chat.ask-for">
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
                    :class="cloudSearchAvailable ? 'text-primary' : 'text-base-content/40'"
                  />
                  <div class="flex min-w-0 flex-1 flex-col">
                    <span class="truncate text-sm" :class="cloudSearchAvailable ? 'text-primary' : ''">
                      <i18n-t scope="global" keypath="cloud-search.search-logs-for">
                        <template #query>
                          <span class="font-mono">{{ query }}</span>
                        </template>
                      </i18n-t>
                    </span>
                    <span class="text-base-content/50 mt-0.5 truncate text-xs">
                      <template v-if="cloudSearchAvailable">
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
                  <template v-if="cloudSearchAvailable && canHover">
                    <kbd class="kbd kbd-xs shrink-0">⇧</kbd>
                    <kbd class="kbd kbd-xs shrink-0">↵</kbd>
                  </template>
                </template>
              </a>
            </li>
          </ul>
        </template>

        <!-- Before anything is typed: the prefixes that narrow the palette. -->
        <div
          v-if="!query && !drilled && canHover"
          class="text-base-content/40 flex items-center gap-3 px-4 pt-3 pb-1.5 text-[11px]"
          data-testid="prefix-hints"
        >
          <span class="flex items-center gap-1">
            <kbd class="kbd kbd-xs">&gt;</kbd> {{ $t("command-palette.prefix-commands") }}
          </span>
          <span class="flex items-center gap-1">
            <kbd class="kbd kbd-xs">@</kbd> {{ $t("command-palette.prefix-hosts") }}
          </span>
          <span v-if="askAvailable" class="flex items-center gap-1">
            <kbd class="kbd kbd-xs">?</kbd> {{ $t("command-palette.prefix-ask") }}
          </span>
        </div>
      </div>
    </div>

    <!-- Footer: what the keys do for the current selection, plus cloud status. -->
    <div
      class="bg-base-300/40 border-base-content/10 text-base-content/50 flex items-center gap-4 border-t px-4 py-2 text-[11px] whitespace-nowrap"
    >
      <!-- One row tall: a hint that does not fit wraps out of sight whole, last
           first, instead of pushing the cloud status off the edge. -->
      <div class="flex h-5 min-w-0 flex-1 flex-wrap content-start gap-x-4 gap-y-5 overflow-hidden *:h-5">
        <span v-if="flatEntries.length > 1" class="hidden items-center gap-1 sm:flex">
          <kbd class="kbd kbd-xs">↑</kbd><kbd class="kbd kbd-xs">↓</kbd>
          <span class="ml-0.5">{{ $t("cloud-search.nav-hint") }}</span>
        </span>
        <span v-if="enterHint" class="hidden items-center gap-1 sm:flex">
          <kbd class="kbd kbd-xs">↵</kbd> <span class="ml-0.5">{{ enterHint }}</span>
        </span>
        <span v-if="selectedContainerItem && !drilled" class="hidden items-center gap-1 md:flex">
          <kbd class="kbd kbd-xs">tab</kbd>
          <span class="ml-0.5">{{ $t("command-palette.actions-hint") }}</span>
        </span>
        <span v-if="drilled" class="hidden items-center gap-1 sm:flex">
          <kbd class="kbd kbd-xs">esc</kbd>
          <span class="ml-0.5">{{ $t("command-palette.back-hint") }}</span>
        </span>
        <!-- Pin and merge only apply to a container, not to a service or stack row. -->
        <span v-if="selectedContainerItem" class="hidden items-center gap-1 md:flex">
          <kbd class="kbd kbd-xs">⌥</kbd><kbd class="kbd kbd-xs">↵</kbd>
          <span class="ml-0.5">{{ $t("cloud-search.pin-shortcut") }}</span>
        </span>
        <span
          v-if="selectedContainerItem && canToggle(selectedContainerItem.id)"
          class="hidden items-center gap-1 md:flex"
        >
          <kbd class="kbd kbd-xs">{{ isMac ? "⌘" : "Ctrl" }}</kbd
          ><kbd class="kbd kbd-xs">↵</kbd>
          <span class="ml-0.5">{{
            isStreaming(selectedContainerItem.id)
              ? $t("cloud-search.unmerge-shortcut")
              : $t("cloud-search.merge-shortcut")
          }}</span>
        </span>
      </div>

      <!-- Cloud status. Skipped while the log search row is on screen, which
           already carries the same call to action. A failed status check wins over
           the config, the same as the cloud icon in the top bar. -->
      <span v-if="cloudConfig?.linked && cloudStatusError" class="ml-auto flex shrink-0 items-center gap-1.5">
        <mdi:cloud-alert-outline
          class="size-3.5"
          :class="cloudStatusError === 'unavailable' ? 'text-warning' : 'text-error'"
        />
        <RouterLink to="/settings/cloud" class="link link-hover" @click.stop>
          {{ $t("cloud-search.cloud-error") }}
        </RouterLink>
      </span>
      <span v-else-if="cloudSearchAvailable" class="ml-auto flex shrink-0 items-center gap-1.5">
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
import { useCommands, useContainerCommands, type Command } from "@/composable/app/commands";
import { Container } from "@/models/Container";
import MdiTextBoxOutline from "~icons/mdi/text-box-outline";
import CilColumns from "~icons/cil/columns";
import PhArrowsMerge from "~icons/ph/arrows-merge";

trackUsage("palette.open");

const emit = defineEmits<{ close: [] }>();
const close = () => emit("close");

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
const { visibleContainers, allContainersById } = storeToRefs(containerStore);

const swarmStore = useSwarmStore();
const { stacks, services } = storeToRefs(swarmStore);

const { cloudConfig, cloudStatusError, ensureCloudStatus } = useCloudConfig();
// A log page on a phone has no cloud icon to have asked already.
ensureCloudStatus();
// Whether the log search row can run. Read from config, not useCloudLogSearch:
// mounting that here ran a Cloud search on every keystroke and threw the hits away.
const cloudSearchAvailable = computed(() => !!cloudConfig.value?.linked && !!cloudConfig.value?.streamLogs);

// A leading character narrows the palette: > commands, @ hosts, ? the assistant.
// Not inside a container's actions, where everything typed filters the actions.
type Scope = "all" | "commands" | "hosts" | "ask";
const PREFIXES: Record<string, Scope> = { ">": "commands", "@": "hosts", "?": "ask" };
// A query reopened from /cloud/search is a log query, so its first character is
// never a prefix until the user edits it.
const scope = computed<Scope>(() => {
  if (drilled.value || (initialQuery && query.value === initialQuery)) return "all";
  return PREFIXES[query.value.trimStart()[0] ?? ""] ?? "all";
});
// What is searched for, without the prefix.
const trimmedQuery = computed(() =>
  scope.value === "all" ? query.value.trim() : query.value.trimStart().slice(1).trim(),
);
const logSearchVisible = computed(() => scope.value === "all" && !drilled.value && trimmedQuery.value.length > 0);

const { t } = useI18n();
const placeholderCopy = computed(() =>
  cloudSearchAvailable.value ? t("cloud-search.modal-placeholder-cloud") : t("cloud-search.modal-placeholder-plain"),
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
  type: "container" | "service" | "stack" | "host";
};

const { hosts } = useHosts();

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

// Hosts are only searched for by name: they never show up as a recent item.
const hostItems = computed<Item[]>(() =>
  Object.values(hosts.value).map((host) => ({
    id: host.id,
    created: new Date(0),
    name: host.name,
    state: host.available ? "running" : "exited",
    type: "host",
  })),
);

const { results: fuseResults } = useFuse(trimmedQuery, list, {
  fuseOptions: {
    keys: ["name", "host"],
    includeScore: true,
    useExtendedSearch: true,
    threshold: 0.3,
    includeMatches: true,
  },
});

const { results: hostFuseResults } = useFuse(trimmedQuery, hostItems, {
  fuseOptions: { keys: ["name"], includeScore: true, threshold: 0.3, includeMatches: true },
  matchAllWhenSearchEmpty: true,
});

// How often and how recently each command ran, so the ones a person reaches
// for come first. Per browser: it is a convenience, not something to sync.
const USAGE_LIMIT = 20;
type Usage = Record<string, { count: number; last: number }>;
const usage = useLocalStorage<Usage>("DOZZLE_PALETTE_USAGE", {});

// A clock set back would put `last` in the future; count that as just now.
const score = (entry?: { count: number; last: number }) =>
  entry ? entry.count / (1 + Math.max(0, Date.now() - entry.last) / 86_400_000) : 0;
const frecency = (id: string) => score(usage.value?.[id]);

function recordUse(id: string) {
  const next: Usage = { ...usage.value, [id]: { count: (usage.value?.[id]?.count ?? 0) + 1, last: Date.now() } };
  const ids = Object.keys(next).sort((a, b) => score(next[b]) - score(next[a]));
  // The command just used always stays, or a new habit could never build up a count.
  for (const stale of ids.filter((other) => other !== id).slice(USAGE_LIMIT - 1)) delete next[stale];
  usage.value = next;
}

// Commands palette. Fuzzy-matched against title/keywords while typing; the
// context commands (container actions) show up front on an empty query.
const { commands, contextCommands, updatable } = useCommands();
const { results: commandFuseResults } = useFuse(trimmedQuery, commands, {
  fuseOptions: {
    keys: ["title", "keywords"],
    includeScore: true,
    useExtendedSearch: true,
    threshold: 0.3,
  },
});

// Matches of about the same quality are ordered by use. A clearly better match
// still wins, so a habit never hides what was typed.
const byUse = (a: Command, b: Command) => frecency(b.id) - frecency(a.id);
const rankedCommandMatches = computed(() =>
  [...commandFuseResults.value]
    .sort((a, b) => Math.round((a.score ?? 0) * 10) - Math.round((b.score ?? 0) * 10) || byUse(a.item, b.item))
    .map((r) => r.item),
);

// With nothing typed: the container actions for this page, then the commands
// this browser used most, so a habit is one keystroke away.
const RECENT_COMMANDS = 3;
const suggestedCommands = computed(() => {
  const context = contextCommands.value;
  const recent = commands.value
    // Review updates has its own row in the updates section.
    .filter((c) => frecency(c.id) > 0 && !context.includes(c) && c.id !== "updates.review")
    .sort(byUse)
    .slice(0, RECENT_COMMANDS);
  return [...context, ...recent];
});

const commandEntries = computed<Command[]>(() => {
  if (scope.value === "commands") {
    return trimmedQuery.value ? rankedCommandMatches.value : [...commands.value].sort(byUse);
  }
  return trimmedQuery.value ? rankedCommandMatches.value : suggestedCommands.value;
});

const containerResults = computed<FuseResult<Item>[]>(() => {
  if (!trimmedQuery.value) return [];
  return [...fuseResults.value, ...hostFuseResults.value].sort((a, b) => {
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

// Containers with an update waiting, offered before anything is typed.
const UPDATES_LIMIT = 5;
const updateResults = computed<FuseResult<Item>[]>(() =>
  updatable.value.slice(0, UPDATES_LIMIT).map((container, refIndex) => ({
    item: {
      id: container.id,
      created: container.created,
      name: container.name,
      state: container.state,
      host: container.hostLabel,
      icon: container.icon,
      type: "container",
    },
    refIndex,
  })),
);

// Empty-query suggestions: running containers first, newest first. Shaped like
// a Fuse result so the rows render through the same branch. A container already
// listed under updates is not repeated.
const recentResults = computed<FuseResult<Item>[]>(() => {
  if (trimmedQuery.value) return [];
  const listed = new Set(updateResults.value.map((r) => r.item.id));
  return [...list.value]
    .filter((item) => !listed.has(item.id))
    .sort((a, b) => {
      if (a.state === b.state) return b.created.getTime() - a.created.getTime();
      return a.state === "running" ? -1 : 1;
    })
    .slice(0, RECENT_LIMIT)
    .map((item, refIndex) => ({ item, refIndex }));
});

const { hasUpdate, isDismissed } = useImageUpdates();
const containerById = (id: string) => allContainersById.value?.[id];
function hasWaitingUpdate(item: Item) {
  const container = item.type === "container" ? containerById(item.id) : undefined;
  return !!container && hasUpdate(container) && !isDismissed(container);
}

// Tab or → on a container lists what can be done to it, from any page.
const drilled = shallowRef<Item>();
let queryBeforeDrill = "";
const drilledContainer = computed<Container | undefined>(() =>
  drilled.value ? allContainersById.value?.[drilled.value.id] : undefined,
);
const drilledCommands = useContainerCommands(drilledContainer);
// A container removed while its actions are open leaves nothing to act on.
watch(drilledContainer, (container) => {
  if (drilled.value && !container) drillOut();
});

const actionCommands = computed<Command[]>(() => {
  const item = drilled.value;
  if (!item || !drilledContainer.value) return [];
  const actions: Command[] = [
    {
      id: "container.open",
      section: "container",
      icon: MdiTextBoxOutline,
      title: t("command-palette.open-logs"),
      keywords: "open logs view",
      perform: () => navigate(item),
    },
    ...drilledCommands.value,
  ];
  if (!isMobile.value) {
    actions.push({
      id: "container.pin",
      section: "container",
      icon: CilColumns,
      title: t("tooltip.pin-column"),
      keywords: "pin column split side",
      perform: () => pinnedStore.pinContainer(item),
    });
  }
  if (canToggle(item.id)) {
    actions.push({
      id: "container.merge",
      section: "container",
      icon: PhArrowsMerge,
      title: toggleTitle(item.id),
      keywords: "merge stream combine add remove",
      perform: () => toggleMerge(item.id),
    });
  }
  // An update waiting is most likely why the container was picked.
  const update = actions.findIndex((c) => c.id === "container.update" && c.detail);
  if (update > 0) actions.splice(1, 0, ...actions.splice(update, 1));
  return actions;
});

const { results: actionFuseResults } = useFuse(trimmedQuery, actionCommands, {
  fuseOptions: { keys: ["title", "keywords"], threshold: 0.3 },
  matchAllWhenSearchEmpty: true,
});

function drillInto(item: Item) {
  queryBeforeDrill = query.value;
  drilled.value = item;
  query.value = "";
  selectedIndex.value = 0;
  // After the render: the crumb and clear button swap in the same patch, and
  // Vue moves the input to do it, which drops its focus.
  nextTick(() => input.value?.focus());
}

function drillOut() {
  const item = drilled.value;
  drilled.value = undefined;
  query.value = queryBeforeDrill;
  // Back on the row the user drilled into, so → then ← is a no-op.
  nextTick(() => {
    input.value?.focus();
    const index = flatEntries.value.findIndex((e) => e.kind === "container" && e.item.id === item?.id);
    selectedIndex.value = Math.max(index, 0);
  });
}

type EntryKind =
  | { kind: "command"; command: Command }
  | { kind: "container"; result: FuseResult<Item>; item: Item }
  | { kind: "logs" }
  | { kind: "ask" };
type PendingEntry = EntryKind & { key: string };
type Entry = EntryKind & { key: string; index: number };
// `count` stands in for the row count when the group holds more than it lists.
type Group = { id: string; label: string; entries: Entry[]; count?: number };

const commandEntry = (command: Command): PendingEntry => ({ kind: "command", command, key: `command:${command.id}` });
const containerEntry = (result: FuseResult<Item>): PendingEntry => ({
  kind: "container",
  result,
  item: result.item,
  key: `${result.item.type}:${result.item.id}`,
});

// One flat, ordered list of everything selectable, sliced into labeled groups.
// Each row carries its position in `flatEntries`, so arrow keys, Enter and
// scroll-into-view all agree without any per-section offset arithmetic.
const groups = computed<Group[]>(() => {
  const result: Group[] = [];
  let index = 0;

  const push = (id: string, label: string, entries: PendingEntry[], count?: number) => {
    if (!entries.length) return;
    result.push({ id, label, count, entries: entries.map((entry) => ({ ...entry, index: index++ })) as Entry[] });
  };

  if (drilled.value) {
    push(
      "actions",
      t("command-palette.actions-hint"),
      actionFuseResults.value.map((r) => commandEntry(r.item)),
    );
    return result;
  }

  if (scope.value === "hosts") {
    push("hosts", t("command-palette.section-hosts"), hostFuseResults.value.map(containerEntry));
    return result;
  }

  if (scope.value === "all") {
    // Containers lead: this is a log viewer, so a typed query is far more often a
    // container name than a command. Enter on the first row opens a container.
    if (!trimmedQuery.value) {
      const review = commands.value.find((c) => c.id === "updates.review");
      // Counts containers, not rows: the review row is not an update, and the
      // list stops at UPDATES_LIMIT.
      push(
        "updates",
        t("command-palette.section-updates"),
        [...updateResults.value.map(containerEntry), ...(review ? [commandEntry(review)] : [])],
        updatable.value.length,
      );
    }
    push(
      "containers",
      trimmedQuery.value ? t("cloud-search.containers-section") : t("command-palette.section-recent"),
      (trimmedQuery.value ? containerResults.value : recentResults.value).map(containerEntry),
    );
  }

  if (scope.value !== "ask") {
    push("commands", t("command-palette.section-commands"), commandEntries.value.map(commandEntry));
  }

  if (scope.value === "all" && logSearchVisible.value) {
    push("logs", t("cloud-search.section-logs"), [{ kind: "logs", key: "logs" }]);
  }

  // Last row, and only with something typed: the palette is for finding
  // things, and asking is what you do when finding did not answer it.
  if (scope.value !== "commands" && askVisible.value) {
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
const askAvailable = computed(() => cloudLinked.value && railAvailable.value);
const askVisible = computed(() => askAvailable.value && !!trimmedQuery.value);
const noLocalMatches = computed(() => {
  if (!trimmedQuery.value) return false;
  if (drilled.value) return !actionFuseResults.value.length;
  if (scope.value === "ask") return !askVisible.value;
  return !flatEntries.value.some((e) => e.kind === "command" || e.kind === "container");
});

// A command that asks for Enter twice, armed by the first press. Moving away
// or typing disarms it, so the second press always lands on what was read.
const armed = ref<string>();
let armedAt = 0;
// Keyed on the row, not its position: a live update can slide another row under
// the same index, and the second Enter must never run that one.
watch([() => selectedEntry.value?.key, query, drilled], () => {
  armed.value = undefined;
});
// A double click is one gesture, so it may arm but never confirm.
const CONFIRM_DELAY = 400;
const isArmed = (entry: Entry) => entry.kind === "command" && armed.value === entry.key;

const enterHint = computed(() => {
  const entry = selectedEntry.value;
  switch (entry?.kind) {
    case "command":
      return isArmed(entry) ? t("command-palette.confirm-hint") : t("command-palette.run-hint");
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
watch(query, () => {
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

function navigate(item: Item) {
  if (item.type === "container") {
    router.push({ name: "/container/[id]", params: { id: item.id } });
  } else if (item.type === "service") {
    router.push({ name: "/service/[name]", params: { name: item.id } });
  } else if (item.type === "stack") {
    router.push({ name: "/stack/[name]", params: { name: item.id } });
  } else if (item.type === "host") {
    router.push({ name: "/host/[id]", params: { id: item.id } });
  }
}

function selected(item: Item) {
  navigate(item);
  close();
}

async function runCommand(command: Command, key: string, clicked = false) {
  if (command.confirm && (armed.value !== key || (clicked && Date.now() - armedAt < CONFIRM_DELAY))) {
    if (armed.value !== key) armedAt = Date.now();
    armed.value = key;
    return;
  }
  recordUse(command.id);
  close();
  await command.perform();
}

function activate(entry: Entry, clicked = false) {
  if (entry.kind === "command") {
    runCommand(entry.command, entry.key, clicked);
  } else if (entry.kind === "container") {
    selected(entry.item);
  } else if (entry.kind === "ask") {
    askAssistant();
  } else {
    runLogSearch();
  }
}

function onEnter(e: KeyboardEvent) {
  // A held key repeats, and an IME uses Enter to commit text: neither is a choice,
  // and a repeat would answer a confirm the user never read.
  if (e.repeat || e.isComposing) return;
  const entry = selectedEntry.value;
  if (entry) activate(entry);
}

// Pin and merge are only meaningful on a container row, not a service, stack,
// host, command or the log search row.
const selectedContainerItem = computed(() => {
  const entry = selectedEntry.value;
  return entry?.kind === "container" && entry.item.type === "container" ? entry.item : undefined;
});

function onTab(e: KeyboardEvent) {
  if (e.isComposing) return;
  const item = selectedContainerItem.value;
  if (!item || drilled.value) return;
  e.preventDefault();
  drillInto(item);
}

// → only drills in with the caret at the end, so it still moves through typed text.
function onRight(e: KeyboardEvent) {
  const el = input.value;
  if (el && el.selectionStart !== el.value.length) return;
  onTab(e);
}

// ← or Backspace at the start of the box goes back out, like a breadcrumb.
function onBack(e: KeyboardEvent) {
  // Vue's .delete matches forward Delete too, which should keep editing the text.
  if (e.key === "Delete" || e.isComposing) return;
  const el = input.value;
  if (!drilled.value || (el && (el.selectionStart !== 0 || el.selectionEnd !== 0))) return;
  e.preventDefault();
  drillOut();
}

// Escape in the action list goes back to the results instead of closing.
function onEscape(e: KeyboardEvent) {
  if (!drilled.value) return;
  e.preventDefault();
  drillOut();
}

function onPin() {
  // Alt+Enter pins a container column.
  if (selectedContainerItem.value) addColumn(selectedContainerItem.value);
}

// Cmd/Ctrl+Enter adds the container to the stream on screen, or takes it out.
function onMerge() {
  const item = selectedContainerItem.value;
  if (item && canToggle(item.id)) mergeItem(item);
}

const { isStreaming, canToggle, toggleTitle, toggle: toggleMerge } = useStreamedContainers();
function mergeItem(item: Item) {
  toggleMerge(item.id);
  close();
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
  if (!cloudSearchAvailable.value) {
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

// Rendered with v-html, and names come from container labels anyone starting a
// container controls, so every slice is escaped before the <mark> tags go in.
function matchedName({ item, matches = [] }: FuseResult<Item>) {
  const matched = matches.find((match) => match.key === "name");
  if (matched) {
    const { indices } = matched;
    const result = [];
    let lastIndex = 0;
    for (const [start, end] of indices) {
      if (lastIndex > start) continue;
      result.push(escapeHtml(item.name.slice(lastIndex, start)));
      result.push(`<mark>${escapeHtml(item.name.slice(start, end + 1))}</mark>`);
      lastIndex = end + 1;
    }
    result.push(escapeHtml(item.name.slice(lastIndex)));
    return result.join("");
  } else {
    return escapeHtml(item.name);
  }
}
</script>

<style scoped>
@reference "@/main.css";
:deep(mark) {
  @apply bg-transparent text-inherit underline underline-offset-2;
}
</style>
