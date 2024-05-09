<template>
  <page-with-links class="gap-16">
    <section>
      <ul class="flex flex-row flex-wrap gap-4">
        <li v-for="host in hostSummaries" class="card w-1/3 bg-base-lighter">
          <div class="card-body flex-row gap-4">
            <div>
              <div class="card-title">{{ host.name }}</div>
              <div class="text-sm">{{ host.containers.length }} containers</div>
              <div class="text-sm">{{ host.nCPU }} CPUs</div>
              <div class="text-sm">{{ formatBytes(host.memTotal) }}</div>
            </div>

            <div
              class="radial-progress text-primary"
              :style="`--value: ${Math.floor((host.avgTotalCPU / (host.nCPU * 100)) * 100)}; --thickness: 0.25em`"
              role="progressbar"
            >
              {{ host.avgTotalCPU.toFixed(0) }}%
            </div>
            <div
              class="radial-progress text-primary"
              :style="`--value: ${(host.avgTotalMem / host.memTotal) * 100}; --thickness: 0.25em`"
              role="progressbar"
            >
              {{ ((host.avgTotalMem / host.memTotal) * 100).toFixed(0) }}%
            </div>
          </div>
        </li>
      </ul>
    </section>
    <!-- <section>
      <div class="stats grid bg-base-lighter shadow">
        <div class="stat">
          <div class="stat-value">{{ runningContainers.length }} / {{ hostContainers.length }}</div>
          <div class="stat-title">{{ $t("label.running") }} / {{ $t("label.total-containers") }}</div>
        </div>
        <div class="stat">
          <div class="stat-figure">
            <div
              class="radial-progress"
              :style="`--value: ${Math.floor(totalCpu) / 2}; --thickness: 0.25em`"
              role="progressbar"
            >
              {{ totalCpu.toFixed(0) }}%
            </div>
          </div>
          <div class="stat-value">8 CPUs</div>
          <div class="stat-title">{{ $t("label.total-cpu-usage") }}</div>
        </div>
        <div class="stat">
          <div class="stat-figure">
            <div
              class="radial-progress"
              :style="`--value: ${Math.floor(totalMem) / 20000000}; --thickness: 0.25em`"
              role="progressbar"
            >
              {{ totalMem.toFixed(0) }}%
            </div>
          </div>
          <div class="stat-value">{{ formatBytes(totalMem) }}</div>
          <div class="stat-title">{{ $t("label.total-mem-usage") }}</div>
        </div>

        <div class="stat">
          <div class="stat-value">{{ Object.keys(hosts).length }}</div>
          <div class="stat-title">{{ $t("label.hosts") }}</div>
          <div class="stat-desc text-secondary">Showing only localhost</div>
        </div>
      </div>
    </section> -->

    <section>
      <container-table :containers="runningContainers"></container-table>
    </section>
  </page-with-links>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";

const { t } = useI18n();
const { hosts } = useHosts();

const containerStore = useContainerStore();
const { containers, ready } = storeToRefs(containerStore) as unknown as {
  containers: Ref<Container[]>;
  ready: Ref<boolean>;
};

type HostSummary = {
  name: string;
  containers: Container[];
  avgTotalCPU: number;
  avgTotalMem: number;
  nCPU: number;
  memTotal: number;
};

const hostSummaries = computed(() => {
  console.log("hostSummaries");
  const summaries: Record<string, HostSummary> = {};
  for (const container of containers.value) {
    if (!summaries[container.host]) {
      const host = hosts.value[container.host];
      summaries[container.host] = reactive({
        name: host.name,
        containers: [],
        avgTotalCPU: 0,
        avgTotalMem: 0,
        nCPU: host.nCPU,
        memTotal: host.memTotal,
      });
    }
    const summary = summaries[container.host];
    summary.containers.push(container);
  }

  return Object.values(summaries).sort((a, b) => a.name.localeCompare(b.name));
});

const hostContainers = $computed(() =>
  containers.value.filter((c) => sessionHost.value === null || c.host === sessionHost.value),
);

const mostRecentContainers = $computed(() => [...hostContainers].sort((a, b) => +b.created - +a.created));
const runningContainers = $computed(() => mostRecentContainers.filter((c) => c.state === "running"));

useIntervalFn(
  () => {
    for (const summary of hostSummaries.value) {
      summary.avgTotalCPU = 0;
      summary.avgTotalMem = 0;
      for (const container of summary.containers) {
        summary.avgTotalCPU += container.movingAverage.cpu;
        summary.avgTotalMem += container.movingAverage.memoryUsage;
      }
    }
  },
  1000,
  { immediate: true },
);

watchEffect(() => {
  if (ready.value) {
    setTitle(t("title.dashboard", { count: runningContainers.length }));
  }
});
</script>
<style lang="postcss" scoped>
:deep(tr td) {
  padding-top: 1em;
  padding-bottom: 1em;
}

.stat > div {
  @apply text-center;
}

.stat-value {
  @apply font-light;
}

.stat-title {
  @apply font-light;
}

.section + .section {
  padding-top: 0;
}
</style>
