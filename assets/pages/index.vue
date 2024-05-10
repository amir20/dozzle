<template>
  <page-with-links class="gap-16">
    <section>
      <ul class="grid grid-cols-[repeat(auto-fill,minmax(480px,1fr))] gap-4">
        <li v-for="host in hostSummaries" class="card bg-base-lighter">
          <div class="4 card-body flex-row justify-between">
            <div>
              <div class="card-title">{{ host.name }}</div>
              <div class="text-sm">{{ host.containers.length }} containers</div>
              <div class="text-sm">{{ host.nCPU }} CPUs</div>
              <div class="text-sm">{{ formatBytes(host.memTotal) }}</div>
            </div>

            <div class="flex flex-row gap-8">
              <div
                class="radial-progress text-primary"
                :style="`--value: ${Math.floor((host.totalCPU / (host.nCPU * 100)) * 100)}; --thickness: 0.25em`"
                role="progressbar"
              >
                {{ host.totalCPU.toFixed(0) }}%
              </div>
              <div
                class="radial-progress text-primary"
                :style="`--value: ${(host.totalMem / host.memTotal) * 100}; --thickness: 0.25em`"
                role="progressbar"
              >
                {{ formatBytes(host.totalMem, 1) }}
              </div>
            </div>
          </div>
        </li>
      </ul>
    </section>

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
  totalCPU: number;
  totalMem: number;
  nCPU: number;
  memTotal: number;
};

const hostSummaries = computed(() => {
  const summaries: Record<string, HostSummary> = {};
  for (const container of containers.value) {
    if (!summaries[container.host]) {
      const host = hosts.value[container.host];
      summaries[container.host] = reactive({
        name: host.name,
        containers: [],
        totalCPU: 0,
        totalMem: 0,
        nCPU: host.nCPU,
        memTotal: host.memTotal,
      });
    }
    const summary = summaries[container.host];
    summary.containers.push(container);
  }

  return Object.values(summaries).sort((a, b) => a.name.localeCompare(b.name));
});

const runningContainers = computed(() => containers.value.filter((c) => c.state === "running"));

useIntervalFn(
  () => {
    for (const summary of hostSummaries.value) {
      summary.totalCPU = 0;
      summary.totalMem = 0;
      for (const container of summary.containers) {
        summary.totalCPU += container.stat.cpu;
        summary.totalMem += container.stat.memoryUsage;
      }
    }
  },
  1000,
  { immediate: true },
);

watchEffect(() => {
  if (ready.value) {
    setTitle(t("title.dashboard", { count: runningContainers.value.length }));
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
