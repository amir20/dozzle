<template>
  <div class="panel">
    <o-autocomplete
      ref="autocomplete"
      v-model="query"
      :placeholder="$t('placeholder.search-containers')"
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
              <octicon:container-24 />
            </span>
          </div>
          <div class="media-content">
            {{ item.name }}
          </div>
          <div class="media-right">
            <span
              class="icon is-small column-icon"
              @click.stop.prevent="addColumn(item)"
              :title="$t('tooltip.pin-column')"
            >
              <cil:columns />
            </span>
          </div>
        </div>
      </template>
    </o-autocomplete>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
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

const list = computed(() => {
  return containers.value.map(({ id, created, name, state }) => {
    return {
      id,
      created,
      name,
      state,
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
    .map(({ item }) => item);
});
watchOnce(autocomplete, () => autocomplete.value?.focus());

function selected({ id }: { id: string }) {
  router.push({ name: "container-id", params: { id } });
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

@media screen and (max-width: 768px) {
  .panel {
    min-height: 200px;
    width: auto;
    margin-left: 0.25rem !important;
    margin-right: 0.25rem !important;
  }
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
