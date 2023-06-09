<template>
  <div>
    <section class="level section pb-0-is-mobile">
      <div class="level-item has-text-centered">
        <div>
          <p class="title">{{ containers.length }}</p>
          <p class="heading">{{ $t("label.total-containers") }}</p>
        </div>
      </div>
      <div class="level-item has-text-centered">
        <div>
          <p class="title">{{ runningContainers.length }}</p>
          <p class="heading">{{ $t("label.running") }}</p>
        </div>
      </div>
      <div class="level-item has-text-centered" data-ci-skip>
        <div>
          <p class="title">{{ totalCpu }}%</p>
          <p class="heading">{{ $t("label.total-cpu-usage") }}</p>
        </div>
      </div>
      <div class="level-item has-text-centered" data-ci-skip>
        <div>
          <p class="title">{{ formatBytes(totalMem) }}</p>
          <p class="heading">{{ $t("label.total-mem-usage") }}</p>
        </div>
      </div>
      <div class="level-item has-text-centered">
        <div>
          <p class="title">{{ version }}</p>
          <p class="heading">{{ $t("label.dozzle-version") }}</p>
        </div>
      </div>
    </section>

    <section class="section table-container">
      <table class="table is-fullwidth">
        <thead>
          <th>Name</th>
          <th>State</th>
          <th>Running</th>
          <th>Avg. CPU</th>
          <th>Avg. Memory</th>
          <th>Actions</th>
        </thead>
        <tbody>
          <tr v-for="container in data" :key="container.id">
            <td>
              {{ container.name }}
            </td>
            <td>
              {{ container.state }}
            </td>
            <td>
              <distance-time :date="container.created" strict :suffix="false"></distance-time>
            </td>
            <td>
              {{ (container.movingAverageStat.cpu / 100).toLocaleString(undefined, { style: "percent" }) }}
            </td>

            <td>
              {{ formatBytes(container.movingAverageStat.memoryUsage) }}
            </td>
            <td>
              <router-link :to="`/containers/${container.id}`" class="button is-small is-primary"> GO </router-link>
            </td>
          </tr>
        </tbody>
      </table>
    </section>
  </div>
</template>

<script lang="ts" setup>
import { useFuse } from "@vueuse/integrations/useFuse";

const { version } = config;
const containerStore = useContainerStore();
const { containers } = storeToRefs(containerStore);
const router = useRouter();

const sort = $ref("running");
const query = ref("");

const mostRecentContainers = $computed(() => [...containers.value].sort((a, b) => +b.created - +a.created));
const runningContainers = $computed(() => mostRecentContainers.filter((c) => c.state === "running"));

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
  fuseOptions: { keys: ["name"] },
  matchAllWhenSearchEmpty: false,
});

const data = computed(() => {
  if (results.value.length) {
    return results.value.map(({ item }) => item);
  }
  switch (sort) {
    case "all":
      return mostRecentContainers;
    case "running":
      return runningContainers;
    default:
      throw `Invalid sort order: ${sort}`;
  }
});

let totalCpu = $ref(0);
useIntervalFn(
  () => {
    totalCpu = runningContainers.reduce((acc, c) => acc + (c.stat?.cpu ?? 0), 0);
  },
  1000,
  { immediate: true },
);

let totalMem = $ref(0);
useIntervalFn(
  () => {
    totalMem = runningContainers.reduce((acc, c) => acc + (c.stat?.memoryUsage ?? 0), 0);
  },
  1000,
  { immediate: true },
);
</script>
<style lang="scss" scoped>
.panel {
  border: 1px solid var(--border-color);

  .panel-block,
  .panel-tabs {
    border-color: var(--border-color);

    .is-active {
      border-color: var(--border-hover-color);
    }

    .name {
      text-overflow: ellipsis;
      white-space: nowrap;
      overflow: hidden;
    }

    .status {
      margin-left: auto;
      white-space: nowrap;
    }
  }
}

@media screen and (max-width: 768px) {
  .pb-0-is-mobile {
    padding-bottom: 0 !important;
  }

  .pt-0-is-mobile {
    padding-top: 0 !important;
  }
}

.icon {
  padding: 10px 3px;
}
</style>
