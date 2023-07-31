!
<template>
  <table class="table is-fullwidth">
    <thead>
      <tr>
        <th>
          <a href="" @click.prevent="sortField = 'name'">{{ $t("label.container-name") }}</a>
        </th>
        <th>{{ $t("label.status") }}</th>
        <th>{{ $t("label.last-started") }}</th>
        <th>
          <a href="" @click.prevent="sortField = 'cpu'">{{ $t("label.avg-cpu") }}</a>
        </th>
        <th>{{ $t("label.avg-mem") }}</th>
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
        <a class="pagination-link" :class="{ 'is-current': i === currentPage }" @click="currentPage = i">{{ i }}</a>
      </li>
    </ul>
  </nav>
</template>

<script setup lang="ts">
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
const sortField: Ref<"cpu" | "mem" | "name" | "created"> = ref("created");
const sortedContainers = computedWithControl(
  () => [containers.length, sortField.value],
  () => {
    console.log("sorting");
    return containers.sort((a, b) => {
      if (sortField.value === "name") {
        return a.name.localeCompare(b.name);
      } else if (sortField.value === "created") {
        return a.created.getTime() - b.created.getTime();
      } else if (sortField.value === "cpu") {
        return a.movingAverage.cpu - b.movingAverage.cpu;
      } else if (sortField.value === "mem") {
        return a.movingAverage.memory - b.movingAverage.memory;
      } else {
        throw new Error("Unknown sort field");
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
</script>

<style lang="scss" scoped></style>
