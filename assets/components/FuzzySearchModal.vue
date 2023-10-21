<template>
  <div class="dropdown dropdown-open w-full">
    <div class="input input-primary flex h-auto items-center">
      <mdi:magnify class="flex h-8 w-8" />
      <input
        tabindex="0"
        class="input input-ghost input-lg flex-1 px-1"
        ref="input"
        @keyup.down="selectedIndex = Math.min(selectedIndex + 1, data.length - 1)"
        @keyup.up="selectedIndex = Math.max(selectedIndex - 1, 0)"
        @keyup.enter.exact="selected(data[selectedIndex])"
        @keyup.alt.enter="addColumn(data[selectedIndex])"
        v-model="query"
        :placeholder="$t('placeholder.search-containers')"
      />
      <mdi:keyboard-esc class="flex" />
    </div>
    <ul tabindex="0" class="menu dropdown-content rounded-box !relative mt-2 w-full bg-base-lighter p-2">
      <li v-for="(item, index) in data">
        <a
          class="grid auto-cols-max grid-cols-[min-content,auto] gap-2 py-4"
          @click.prevent="selected(item)"
          @mouseenter="selectedIndex = index"
          :class="index === selectedIndex ? 'focus' : ''"
        >
          <div :class="{ 'text-primary': item.state === 'running' }">
            <octicon:container-24 />
          </div>
          <div class="truncate">
            <span class="font-light">{{ item.host }}</span> / {{ item.name }}
          </div>
          <distance-time :date="item.created" class="ml-auto text-xs font-light" />
          <a @click.stop.prevent="addColumn(item)" :title="$t('tooltip.pin-column')" class="hover:text-secondary">
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

const { maxResults: resultLimit = 5 } = defineProps<{
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
  return containers.value.map(({ id, created, name, state, hostLabel: host }) => {
    return {
      id,
      created,
      name,
      state,
      host,
    };
  });
});

const { results } = useFuse(query, list, {
  fuseOptions: {
    keys: ["name", "host"],
    includeScore: true,
    useExtendedSearch: true,
  },
  resultLimit,
  matchAllWhenSearchEmpty: true,
});

const data = computed(() => {
  return results.value
    .sort((a, b) => {
      if (a.score === b.score) {
        if (a.item.state === "running" && b.item.state !== "running") {
          return -1;
        } else {
          return 1;
        }
      } else if (a.score && b.score) {
        return a.score - b.score;
      } else {
        return 0;
      }
    })
    .map(({ item }) => item)
    .slice(0, resultLimit);
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
</script>

<style scoped lang="postcss"></style>
