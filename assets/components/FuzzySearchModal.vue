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
    <ul tabindex="0" class="menu dropdown-content !relative mt-2 w-full rounded-box bg-base-lighter p-2">
      <li v-for="(result, index) in data">
        <a
          class="grid auto-cols-max grid-cols-[min-content,auto] gap-2 py-4"
          @click.prevent="selected(result.item)"
          @mouseenter="selectedIndex = index"
          :class="index === selectedIndex ? 'focus' : ''"
        >
          <div :class="{ 'text-primary': result.item.state === 'running' }">
            <octicon:container-24 />
          </div>
          <div class="truncate">
            <template v-if="config.hosts.length > 1">
              <span class="font-light">{{ result.item.host }}</span> /
            </template>
            <span data-name v-html="matchedName(result)"></span>
          </div>
          <distance-time :date="result.item.created" class="text-xs font-light" />
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
</template>

<script lang="ts" setup>
import { useFuse } from "@vueuse/integrations/useFuse";
import { type FuseResultMatch } from "fuse.js";

const { maxResults = 5 } = defineProps<{
  maxResults?: number;
}>();

const close = defineEmit();

const query = ref("");
const input = ref<HTMLInputElement>();
const selectedIndex = ref(0);

const router = useRouter();
const store = useContainerStore();
const { containers } = storeToRefs(store);

const list = computed(() => {
  return containers.value.map(({ id, created, name, state, labels, hostLabel: host }) => {
    return {
      id,
      created,
      name,
      state,
      host,
      labels: Object.entries(labels).map(([_, value]) => value),
    };
  });
});

const { results } = useFuse(query, list, {
  fuseOptions: {
    keys: ["name", "host", "labels"],
    includeScore: true,
    useExtendedSearch: true,
    threshold: 0.3,
    includeMatches: true,
  },
  resultLimit: 10,
  matchAllWhenSearchEmpty: true,
});

const data = computed(() => {
  return results.value
    .toSorted((a, b) => {
      if (a.score === b.score) {
        if (a.item.state === b.item.state) {
          return b.item.created - a.item.created;
        } else if (a.item.state === "running" && b.item.state !== "running") {
          return -1;
        } else {
          return 1;
        }
      } else {
        return a.score - b.score;
      }
    })
    .slice(0, maxResults);
});

watch(query, (data) => {
  if (data.length > 0) {
    selectedIndex.value = 0;
  }
});

onMounted(() => input.value?.focus());

function selected({ id }: { id: string }) {
  router.push({ name: "container-id", params: { id } });
  close();
}
function addColumn(container: { id: string }) {
  store.appendActiveContainer(container);
  close();
}

function matchedName({ item, matches = [] }: { item: { name: string }; matches?: FuseResultMatch[] }) {
  const matched = matches.find((match) => match.key === "name");
  if (matched) {
    const { indices } = matched;
    const result = [];
    let lastIndex = 0;
    for (const [start, end] of indices) {
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

.menu a {
  @apply transition-none duration-0;
}
</style>
