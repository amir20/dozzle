<template>
  <div class="panel">
    <o-autocomplete
      ref="autocomplete"
      v-model="query"
      placeholder="Search containers using ⌘ + k or ctrl + k"
      field="name"
      open-on-focus
      keep-first
      expanded
      :data="results"
      @select="selected"
    >
      <template #default="props">
        <div class="media">
          <div class="media-left">
            <span class="icon is-small" :class="props.option.state">
              <octicon-container-24 />
            </span>
          </div>
          <div class="media-content">
            {{ props.option.name }}
          </div>
          <div class="media-right">
            <span class="icon is-small column-icon" @click.stop.prevent="addColumn(props.option)" title="Pin as column">
              <cil-columns />
            </span>
          </div>
        </div>
      </template>
    </o-autocomplete>
  </div>
</template>

<script lang="ts" setup>
import fuzzysort from "fuzzysort";
import { type Container } from "@/types/Container";

const props = defineProps({
  maxResults: {
    default: 20,
    type: Number,
  },
});

const emit = defineEmits(["close"]);

const query = ref("");
const autocomplete = ref<HTMLElement>();
const router = useRouter();
const store = useContainerStore();
const { containers } = storeToRefs(store);
const preparedContainers = computed(() =>
  containers.value.map(({ name, id, created, state }) =>
    reactive({
      name,
      id,
      created,
      state,
      preparedName: fuzzysort.prepare(name),
    })
  )
);

const results = computed(() => {
  const options = {
    limit: props.maxResults,
    key: "preparedName",
  };
  if (query.value) {
    const results = fuzzysort.go(query.value, preparedContainers.value, options);
    results.forEach((result) => {
      if (result.obj.state === "running") {
        // @ts-ignore
        result.score += 1;
      }
    });
    return [...results].sort((a, b) => b.score - a.score).map((i) => i.obj);
  } else {
    return [...preparedContainers.value].sort((a, b) => b.created - a.created);
  }
});

onMounted(() => nextTick(() => autocomplete.value?.focus()));

function selected(item: { id: string; name: string }) {
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
