<template>
  <div class="panel">
    <o-autocomplete
      ref="autocomplete"
      v-model="query"
      placeholder="Search containers using ⌘ + k or ctrl + k"
      open-on-focus
      keep-first
      expanded
      :data="data"
      @select="selected"
    >
      <template #default="{ option: item }">
        <div class="media">
          <div class="media-left">
            <span class="icon is-small" :class="item.state">
              <octicon-container-24 />
            </span>
          </div>
          <div class="media-content">
            {{ item.name }}
          </div>
          <div class="media-right">
            <span class="icon is-small column-icon" @click.stop.prevent="addColumn(item)" title="Pin as column">
              <cil-columns />
            </span>
          </div>
        </div>
      </template>
    </o-autocomplete>
  </div>
</template>

<script lang="ts" setup>
import { type Container } from "@/types/Container";
import { useFuse } from "@vueuse/integrations/useFuse";

const { maxResults: resultLimit = 20 } = defineProps<{
  maxResults?: number;
}>();

const emit = defineEmits<{
  (e: "close"): void;
}>();

const query = ref("");
const autocomplete = ref<HTMLElement>();
const router = useRouter();
const store = useContainerStore();
const { containers } = storeToRefs(store);

const { results } = useFuse(query, containers, {
  fuseOptions: { keys: ["name"] },
  resultLimit,
  matchAllWhenSearchEmpty: true,
});

const data = computed(() => results.value.map(({ item }) => item));
watchOnce(autocomplete, () => autocomplete.value?.focus());

function selected({ item }: { item: { id: string; name: string } }) {
  router.push({ name: "container-id", params: { id: item.id } });
  emit("close");
}
function addColumn(container: Container) {
  store.appendActiveContainer(container);
  emit("close");
}
</script>

<style lang="scss" scoped>
.panel {
  min-height: 400px;
  width: 580px;
}

.running {
  color: var(--primary-color);
}

.exited {
  color: var(--scheme-main-ter);
}

.column-icon {
  &:hover {
    color: var(--secondary-color);
  }
}

:deep(a.dropdown-item) {
  padding-right: 1em;
  .media-right {
    visibility: hidden;
  }
  &:hover .media-right {
    visibility: visible;
  }
}

.icon {
  vertical-align: middle;
}
</style>
