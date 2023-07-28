!
<template>
  <table class="table is-fullwidth">
    <thead>
      <tr>
        <th>{{ $t("label.container-name") }}</th>
        <th>{{ $t("label.status") }}</th>
        <th>{{ $t("label.last-started") }}</th>
        <th>{{ $t("label.avg-cpu") }}</th>
        <th>{{ $t("label.avg-mem") }}</th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="container in containers" :key="container.id">
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
</template>

<script setup lang="ts">
const { containers } = defineProps<{
  containers: {
    movingAverage: { cpu: number; memory: number };
    created: Date;
    state: string;
    name: string;
    id: string;
  }[];
}>();
</script>

<style lang="scss" scoped></style>
