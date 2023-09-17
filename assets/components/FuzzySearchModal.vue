<template>
  <div class="dropdown w-full">
    <input
      tabindex="0"
      class="input input-primary w-full bg-base-lighter"
      autofocus
      v-model="query"
      :placeholder="$t('placeholder.search-containers')"
    />
    <ul tabindex="0" class="menu dropdown-content rounded-box !relative mt-2 w-full bg-base-lighter p-2">
      <li v-for="item in data">
        <a class="flex gap-2" @click.prevent="selected(item)">
          <div>
            <octicon:container-24 />
          </div>
          <div>{{ item.host }} / {{ item.name }}</div>
          <a
            @click.stop.prevent="addColumn(item)"
            :title="$t('tooltip.pin-column')"
            class="ml-auto hover:text-secondary"
          >
            <cil:columns />
          </a>
        </a>
      </li>
    </ul>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { useFuse } from "@vueuse/integrations/useFuse";

const { maxResults: resultLimit = 15 } = defineProps<{
  maxResults?: number;
}>();

const close = defineEmit();

const query = ref("");
const autocomplete = ref<HTMLElement>();
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
  fuseOptions: { keys: ["name"], includeScore: true },
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
watchOnce(autocomplete, () => autocomplete.value?.focus());

function selected({ id }: { id: string }) {
  router.push({ name: "container-id", params: { id } });
  query.value = "";
  close();
}
function addColumn(container: Container) {
  store.appendActiveContainer(container);
  query.value = "";
  close();
}
</script>

<style scoped>
.running {
  color: var(--primary-color);
}

.exited {
  color: var(--scheme-main-ter);
}
</style>
