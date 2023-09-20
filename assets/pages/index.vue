<template>
  <div class="flex flex-col gap-16 px-8 pt-8">
    <section>
      <div class="bg-scheme stats grid shadow">
        <div class="stat">
          <div class="stat-value">{{ runningContainers.length }} / {{ containers.length }}</div>
          <div class="stat-title">{{ $t("label.running") }} / {{ $t("label.total-containers") }}</div>
        </div>
        <div class="stat">
          <div class="stat-value">{{ totalCpu }}%</div>
          <div class="stat-title">{{ $t("label.total-cpu-usage") }}</div>
        </div>
        <div class="stat">
          <div class="stat-value">{{ formatBytes(totalMem) }}</div>
          <div class="stat-title">{{ $t("label.total-mem-usage") }}</div>
        </div>

        <div class="stat">
          <div class="stat-value">{{ version }}</div>
          <div class="stat-title">{{ $t("label.dozzle-version") }}</div>
        </div>
      </div>
    </section>

    <section>
      <div class="box">
        <container-table :containers="runningContainers"></container-table>
      </div>
    </section>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";

const { t } = useI18n();
const { version } = config;
const containerStore = useContainerStore();
const { containers, ready } = storeToRefs(containerStore) as unknown as {
  containers: Ref<Container[]>;
  ready: Ref<boolean>;
};

const mostRecentContainers = $computed(() => [...containers.value].sort((a, b) => +b.created - +a.created));
const runningContainers = $computed(() => mostRecentContainers.filter((c) => c.state === "running"));

let totalCpu = $ref(0);
useIntervalFn(
  () => {
    totalCpu = runningContainers.reduce((acc, c) => acc + c.stat.cpu, 0);
  },
  1000,
  { immediate: true },
);

let totalMem = $ref(0);
useIntervalFn(
  () => {
    totalMem = runningContainers.reduce((acc, c) => acc + c.stat.memoryUsage, 0);
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
