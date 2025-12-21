<template>
  <div class="card bg-base-100">
    <div class="card-body flex gap-2">
      <div class="flex flex-col gap-2 overflow-hidden">
        <div class="flex items-center gap-1 truncate text-xl font-semibold">
          <HostIcon :type="host.type" class="flex-none" />
          <div class="truncate">
            {{ host.name }}
          </div>

          <span class="badge badge-error badge-xs gap-2 p-2" v-if="!host.available">
            <carbon:warning />
            offline
          </span>
          <span
            class="badge badge-success badge-xs gap-2 p-2"
            :class="{ 'badge-warning': config.version != host.agentVersion }"
            v-else-if="host.type == 'agent'"
            title="Dozzle Agent"
          >
            {{ host.agentVersion }}
          </span>
        </div>
        <!-- <ul class="flex flex-row gap-x-2 text-sm md:gap-3">
          <li class="flex items-center gap-1"><ph:cpu /> {{ host.nCPU }} <span class="max-md:hidden">CPUs</span></li>
          <li class="flex items-center gap-1">
            <ph:memory /> {{ formatBytes(host.memTotal) }}
            <span class="max-md:hidden">total</span>
          </li>
        </ul> -->
        <ul class="flex flex-row flex-wrap gap-x-2 text-sm md:gap-3">
          <li class="flex items-center gap-1">
            <octicon:container-24 class="inline-block" />
            {{ $t("label.container", containerCount) }}
          </li>
          <li class="flex items-center gap-1"><mdi:docker class="inline-block" /> {{ host.dockerVersion }}</li>
        </ul>
      </div>

      <div class="grid grid-cols-2 gap-3 md:gap-4" v-if="stats">
        <!-- CPU Card -->
        <div class="border-primary/30 bg-primary/10 rounded-lg border p-3">
          <div class="text-primary mb-2 flex items-center gap-1.5 text-xs font-medium">
            <ph:cpu class="text-sm" />
            <span>CPU</span>
          </div>
          <div class="mb-1.5 text-lg font-semibold">4%</div>
          <div class="text-base-content/60 mb-1 text-[10px]">avg 1.2 • pk 4.5</div>
          <!-- Bar chart placeholder -->
          <div class="flex h-8 items-end gap-[2px]">
            <div
              v-for="i in 24"
              :key="i"
              class="bg-primary/50 flex-1 rounded-sm"
              :style="`height: ${20 + Math.random() * 60}%`"
            ></div>
          </div>
        </div>

        <!-- Memory Card -->
        <div class="border-secondary/30 bg-secondary/10 rounded-lg border p-3">
          <div class="text-secondary mb-2 flex items-center gap-1.5 text-xs font-medium">
            <ph:memory class="text-sm" />
            <span>MEM</span>
          </div>
          <div class="mb-1.5 text-lg font-semibold">1.9G</div>
          <div class="text-base-content/60 mb-1 text-[10px]">avg 1.5 • pk 2.1</div>
          <!-- Bar chart placeholder -->
          <div class="flex h-8 items-end gap-[2px]">
            <div
              v-for="i in 24"
              :key="i"
              class="bg-secondary/50 flex-1 rounded-sm"
              :style="`height: ${30 + Math.random() * 50}%`"
            ></div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Host } from "@/stores/hosts";
import { Container } from "@/models/Container";

const props = defineProps<{
  host: Host;
}>();

const containerStore = useContainerStore();
const { containers } = storeToRefs(containerStore) as unknown as {
  containers: Ref<Container[]>;
};

const hostContainers = computed(() =>
  containers.value.filter((container) => container.host === props.host.id && container.state === "running"),
);

const containerCount = computed(() => hostContainers.value.length);

type TotalStat = {
  totalCPU: number;
  totalMem: number;
};

const mostRecent = ref<TotalStat>({ totalCPU: 0, totalMem: 0 });
const stats = reactive({ mostRecent, weighted: useExponentialMovingAverage(mostRecent) });

useIntervalFn(
  () => {
    const stat = { totalCPU: 0, totalMem: 0 };
    for (const container of hostContainers.value) {
      stat.totalCPU += container.stat.cpu;
      stat.totalMem += container.stat.memoryUsage;
    }
    mostRecent.value = stat;
  },
  1000,
  { immediate: true },
);
</script>
