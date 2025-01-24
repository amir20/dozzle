<template>
  <div v-if="ready" data-testid="side-menu" class="flex min-h-0 flex-col">
    <Carousel v-model="selectedCard" class="flex-1">
      <CarouselItem title="Hosts and Containers" id="host">
        <HostMenu />
      </CarouselItem>
      <CarouselItem title="Services and Stacks" v-if="services.length > 0 || customGroups.length > 0" id="swarm">
        <SwarmMenu />
      </CarouselItem>
    </Carousel>
  </div>
  <div role="status" class="flex animate-pulse flex-col gap-4" v-else>
    <div class="bg-base-content/50 h-3 w-full rounded-full opacity-50" v-for="_ in 9"></div>
    <span class="sr-only">Loading...</span>
  </div>
</template>

<script lang="ts" setup>
const containerStore = useContainerStore();
const { ready } = storeToRefs(containerStore);
const route = useRoute();
const swarmStore = useSwarmStore();
const { services, customGroups } = storeToRefs(swarmStore);
const selectedCard = ref<"host" | "swarm">("host");

watch(
  route,
  () => {
    if (route.meta.swarmMode) {
      selectedCard.value = "swarm";
    } else if (route.meta.containerMode) {
      selectedCard.value = "host";
    }
  },
  { immediate: true },
);
</script>
<style scoped></style>
