!
<template>
  <div class="text-right" v-if="containers.length > pageSizes[0]">
    Show per page
    <dropdown-menu
      class="dropdown-left btn-xs md:btn-sm"
      v-model="perPage"
      :options="pageSizes.map((i) => ({ label: i.toLocaleString(), value: i }))"
    />
  </div>
  <table class="table table-lg bg-base">
    <thead>
      <tr :data-direction="direction > 0 ? 'asc' : 'desc'">
        <th
          v-for="(value, key) in fields"
          :key="key"
          @click.prevent="sort(key)"
          :class="{ 'selected-sort': key === sortField }"
          v-show="isVisible(key)"
        >
          <a class="inline-flex cursor-pointer gap-2 text-sm uppercase">
            <span>{{ $t(value.label) }}</span>
            <span class="h-4" data-icon>
              <mdi:arrow-up />
            </span>
          </a>
        </th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="container in paginated" :key="container.id">
        <td v-if="isVisible('name')">
          <router-link :to="{ name: 'container-id', params: { id: container.id } }" :title="container.name">
            {{ container.name }}
          </router-link>
        </td>
        <td v-if="isVisible('host')">{{ container.hostLabel }}</td>
        <td v-if="isVisible('state')">{{ container.state }}</td>
        <td v-if="isVisible('created')">
          <distance-time :date="container.created" strict :suffix="false"></distance-time>
        </td>
        <td v-if="isVisible('cpu')">
          <bar-chart :value="container.movingAverage.cpu / 100">
            {{ (container.movingAverage.cpu / 100).toLocaleString(undefined, { style: "percent" }) }}
          </bar-chart>
        </td>
        <td v-if="isVisible('mem')">
          <bar-chart :value="container.movingAverage.memory / 100">
            {{ (container.movingAverage.memory / 100).toLocaleString(undefined, { style: "percent" }) }}
          </bar-chart>
        </td>
      </tr>
    </tbody>
  </table>
  <div class="p-4 text-center">
    <nav class="join" v-if="isPaginated">
      <button
        v-for="i in totalPages"
        class="btn join-item"
        :class="{ 'btn-primary': i === currentPage }"
        @click="currentPage = i"
      >
        {{ i }}
      </button>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { Container } from "@/models/Container";
import { toRefs } from "@vueuse/core";

const fields = {
  name: {
    label: "label.container-name",
    sortFunc: (a: Container, b: Container) => a.name.localeCompare(b.name) * direction.value,
    mobileVisible: true,
  },
  host: {
    label: "label.host",
    sortFunc: (a: Container, b: Container) => a.hostLabel.localeCompare(b.hostLabel) * direction.value,
    mobileVisible: false,
  },
  state: {
    label: "label.status",
    sortFunc: (a: Container, b: Container) => a.state.localeCompare(b.state) * direction.value,
    mobileVisible: false,
  },
  created: {
    label: "label.created",
    sortFunc: (a: Container, b: Container) => (a.created.getTime() - b.created.getTime()) * direction.value,
    mobileVisible: true,
  },
  cpu: {
    label: "label.avg-cpu",
    sortFunc: (a: Container, b: Container) => (a.movingAverage.cpu - b.movingAverage.cpu) * direction.value,
    mobileVisible: false,
  },
  mem: {
    label: "label.avg-mem",
    sortFunc: (a: Container, b: Container) => (a.movingAverage.memory - b.movingAverage.memory) * direction.value,
    mobileVisible: false,
  },
};

const { containers } = defineProps<{
  containers: Container[];
}>();
type keys = keyof typeof fields;

const perPage = useStorage("DOZZLE_TABLE_PAGE_SIZE", 15);
const pageSizes = [15, 30, 50, 100];

const storage = useStorage<{ column: keys; direction: 1 | -1 }>("DOZZLE_TABLE_CONTAINERS_SORT", {
  column: "created",
  direction: -1,
});
const { column: sortField, direction } = toRefs(storage);
const sortedContainers = computedWithControl(
  () => [containers.length, sortField.value, direction.value],
  () => {
    return containers.sort((a, b) => {
      return fields[sortField.value].sortFunc(a, b);
    });
  },
);

const totalPages = computed(() => Math.ceil(sortedContainers.value.length / perPage.value));
const isPaginated = computed(() => totalPages.value > 1);
const currentPage = ref(1);
watch(perPage, () => (currentPage.value = 1));
const paginated = computed(() => {
  const start = (currentPage.value - 1) * perPage.value;
  const end = start + perPage.value;

  return sortedContainers.value.slice(start, end);
});

function sort(field: keys) {
  if (sortField.value === field) {
    direction.value *= -1;
  } else {
    sortField.value = field;
    direction.value = 1;
  }
}
function isVisible(field: keys) {
  return fields[field].mobileVisible || !isMobile.value;
}
</script>

<style lang="postcss" scoped>
[data-icon] {
  display: none;
  transition: transform 0.2s ease-in-out;
  [data-direction="desc"] & {
    transform: rotate(180deg);
  }
}

th {
  @apply border-b-2 border-base-lighter;
  &.selected-sort {
    font-weight: bold;
    @apply border-primary;
    [data-icon] {
      display: inline-block;
    }
  }
}

tbody td {
  max-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

a {
  @apply hover:text-primary;
}
</style>
