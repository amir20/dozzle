!
<template>
  <table class="table is-fullwidth">
    <thead>
      <tr :data-direction="direction > 0 ? 'asc' : 'desc'">
        <th
          v-for="(label, field) in headers"
          :key="field"
          @click.prevent="sort(field)"
          :class="{ 'selected-sort': field === sortField }"
        >
          <a>
            <span class="icon-text">
              <span>{{ $t(label) }}</span>
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
        <td>
          <router-link :to="{ name: 'container-id', params: { id: container.id } }" :title="container.name">
            {{ container.name }}
          </router-link>
        </td>
        <td>{{ container.state }}</td>
        <td><distance-time :date="container.created" strict :suffix="false"></distance-time></td>
        <td>
          {{ (container.movingAverage.cpu / 100).toLocaleString(undefined, { style: "percent" }) }}
        </td>
        <td>
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
const headers = {
  name: "label.container-name",
  state: "label.status",
  created: "label.last-started",
  cpu: "label.avg-cpu",
  mem: "label.avg-mem",
};
const { containers, perPage = 15 } = defineProps<{
  containers: {
    movingAverage: { cpu: number; memory: number };
    created: Date;
    state: string;
    name: string;
    id: string;
  }[];
  perPage?: number;
}>();
const sortField: Ref<keyof typeof headers> = ref("created");
const direction = ref<1 | -1>(-1);
const sortedContainers = computedWithControl(
  () => [containers.length, sortField.value, direction.value],
  () => {
    return containers.sort((a, b) => {
      switch (sortField.value) {
        case "name":
          return a.name.localeCompare(b.name) * direction.value;
        case "state":
          return a.state.localeCompare(b.state) * direction.value;
        case "created":
          return (a.created.getTime() - b.created.getTime()) * direction.value;
        case "cpu":
          return (a.movingAverage.cpu - b.movingAverage.cpu) * direction.value;
        case "mem":
          return (a.movingAverage.memory - b.movingAverage.memory) * direction.value;
      }
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

function sort(field: keyof typeof headers) {
  if (sortField.value === field) {
    direction.value *= -1;
  } else {
    sortField.value = field;
    direction.value = 1;
  }
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
</style>
