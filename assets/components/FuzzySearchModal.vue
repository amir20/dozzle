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
        @keydown.enter.exact="selected(data[selectedIndex].item)"
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
      v-if="results.length"
    >
      <ul class="menu w-auto">
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
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ContainerState } from "@/types/Container";
import { useFuse } from "@vueuse/integrations/useFuse";
import { type FuseResult } from "fuse.js";

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

const { results } = useFuse(query, list, {
  fuseOptions: {
    keys: ["name", "host"],
    includeScore: true,
    useExtendedSearch: true,
    threshold: 0.3,
    includeMatches: true,
  },
});

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
