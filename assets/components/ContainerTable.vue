!
<template>
  <table class="table is-fullwidth">
    <thead>
      <tr :data-direction="direction > 0 ? 'asc' : 'desc'">
        <th
          v-for="(value, key) in fields"
          :key="key"
          @click.prevent="sort(key)"
          :class="{ 'selected-sort': key === sortField }"
          v-show="isVisible(key)"
        >
          <a>
            <span class="icon-text">
              <span>{{ $t(value.label) }}</span>
              <span class="icon">
                <mdi:arrow-up />
              </span>
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
        <td v-if="isVisible('state')">{{ container.state }}</td>
        <td v-if="isVisible('created')">
          <distance-time :date="container.created" strict :suffix="false"></distance-time>
        </td>
        <td v-if="isVisible('cpu')">
          {{ (container.movingAverage.cpu / 100).toLocaleString(undefined, { style: "percent" }) }}
        </td>
        <td v-if="isVisible('mem')">
          {{ (container.movingAverage.memory / 100).toLocaleString(undefined, { style: "percent" }) }}
        </td>
      </tr>
    </tbody>
  </table>
  <nav class="pagination is-right" role="navigation" aria-label="pagination" v-if="isPaginated">
    <ul class="pagination-list">
      <li v-for="i in totalPages">
        <a class="pagination-link" :class="{ 'is-current': i === currentPage }" @click.prevent="currentPage = i">{{
          i
        }}</a>
      </li>
    </ul>
  </nav>
</template>

<script setup lang="ts">
const fields = {
  name: {
    label: "label.container-name",
    sortFunc: (a: Container, b: Container) => a.name.localeCompare(b.name) * direction.value,
    mobileVisible: true,
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
type Container = {
  id: string;
  name: string;
  state: string;
  created: Date;
  movingAverage: {
    cpu: number;
    memory: number;
  };
};

const { containers, perPage = 15 } = defineProps<{
  containers: Container[];
  perPage?: number;
}>();
const sortField: Ref<keyof typeof fields> = ref("created");
const direction = ref<1 | -1>(-1);
const sortedContainers = computedWithControl(
  () => [containers.length, sortField.value, direction.value],
  () => {
    return containers.sort((a, b) => {
      return fields[sortField.value].sortFunc(a, b);
    });
  },
);

const totalPages = computed(() => Math.ceil(sortedContainers.value.length / perPage));
const isPaginated = computed(() => totalPages.value > 1);
const currentPage = ref(1);
const paginated = computed(() => {
  const start = (currentPage.value - 1) * perPage;
  const end = start + perPage;

  return sortedContainers.value.slice(start, end);
});

function sort(field: keyof typeof fields) {
  if (sortField.value === field) {
    direction.value *= -1;
  } else {
    sortField.value = field;
    direction.value = 1;
  }
}
function isVisible(field: keyof typeof fields) {
  return fields[field].mobileVisible || !isMobile.value;
}
</script>

<style lang="scss" scoped>
.icon {
  display: none;
  transition: transform 0.2s ease-in-out;
  [data-direction="desc"] & {
    transform: rotate(180deg);
  }
}
.selected-sort {
  font-weight: bold;
  border-color: var(--primary-color);
  .icon {
    display: inline-block;
  }
}

tbody td {
  max-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
