<template>
  <PageWithLinks>
    <section class="flex h-[calc(100vh-8rem)] flex-col gap-4">
      <div>
        <h2 class="text-2xl font-bold">Dependency Map</h2>
        <p class="text-base-content/60">Compose stacks with depends_on and shared network relationships</p>
      </div>
      <DependencyGraph v-if="ready" :containers="graphContainers" class="flex-1" />
      <div v-else class="flex flex-1 items-center justify-center">
        <span class="loading loading-spinner loading-lg"></span>
      </div>
    </section>
  </PageWithLinks>
</template>

<script lang="ts" setup>
const containerStore = useContainerStore();
const { containers, ready } = storeToRefs(containerStore);
const graphContainers = computed(() => containers.value.filter((c) => c.state !== "deleted"));
</script>
