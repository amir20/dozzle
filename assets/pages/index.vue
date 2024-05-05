<template>
  <page-with-links class="gap-16">
    <section>
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

const hostContainers = $computed(() =>
  containers.value.filter((c) => sessionHost.value === null || c.host === sessionHost.value),
);

const mostRecentContainers = $computed(() => [...hostContainers].sort((a, b) => +b.created - +a.created));
const runningContainers = $computed(() => mostRecentContainers.filter((c) => c.state === "running"));

let totalCpu = $ref(0);
useIntervalFn(
  () => {
    totalCpu = runningContainers.reduce((acc, c) => acc + c.movingAverage.cpu, 0);
  },
  1000,
  { immediate: true },
);

let totalMem = $ref(0);
useIntervalFn(
  () => {
    totalMem = runningContainers.reduce((acc, c) => acc + c.movingAverage.memoryUsage, 0);
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
