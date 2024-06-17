<template>
  <div class="dropdown dropdown-open w-full">
    <div class="input input-primary flex h-auto items-center">
      <mdi:magnify class="flex size-8" />
      <input
        tabindex="0"
        class="input input-lg input-ghost flex-1 px-1"
        ref="input"
        @keydown.down="selectedIndex = Math.min(selectedIndex + 1, data.length - 1)"
        @keydown.up="selectedIndex = Math.max(selectedIndex - 1, 0)"
        @keydown.enter.exact="selected(data[selectedIndex].item)"
        @keydown.alt.enter="addColumn(data[selectedIndex])"
        v-model="query"
        :placeholder="$t('placeholder.search-containers')"
      />
      <mdi:keyboard-esc class="flex" />
    </div>
    <div
      class="dropdown-content !relative mt-2 max-h-[calc(100dvh-20rem)] w-full overflow-y-scroll rounded-md border-y-8 border-transparent bg-base-lighter px-2"
      v-if="results.length"
    >
      <ul tabindex="0" class="menu">
        <li v-for="(result, index) in data" ref="listItems">
          <a
            class="grid auto-cols-max grid-cols-[min-content,auto] gap-2 py-4"
            @click.prevent="selected(result.item)"
            :class="index === selectedIndex ? 'focus' : ''"
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

            <DistanceTime :date="result.item.created" class="text-xs font-light" />
            <a
              @click.stop.prevent="addColumn(result.item)"
              :title="$t('tooltip.pin-column')"
              class="hover:text-secondary"
            >
              <ic:sharp-keyboard-return v-if="index === selectedIndex" />
              <cil:columns v-else />
            </a>
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

useFocus(input, { initialValue: true });

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

<style scoped lang="postcss">
:deep(mark) {
  @apply bg-transparent text-inherit underline underline-offset-2;
}
</style>
