<template>
  <div v-if="ready" data-testid="side-menu">
    <Toggle v-model="showSwarm" v-if="services.length > 0 || customGroups.length > 0">
      <div class="text-lg font-light">{{ $t("label.swarm-mode") }}</div>
    </Toggle>

    <Carousel>
      <CarouselItem title="Hosts and Containers">
        <HostMenu />
      </CarouselItem>
      <CarouselItem title="Services and Stacks">
        <SwarmMenu />
      </CarouselItem>
    </Carousel>
  </div>
  <div role="status" class="flex animate-pulse flex-col gap-4" v-else>
    <div class="h-3 w-full rounded-full bg-base-content/50 opacity-50" v-for="_ in 9"></div>
    <span class="sr-only">Loading...</span>
  </div>
</template>

<script lang="ts" setup>
const containerStore = useContainerStore();
const { ready } = storeToRefs(containerStore);
const route = useRoute();
const swarmStore = useSwarmStore();
const { services, customGroups } = storeToRefs(swarmStore);

const showSwarm = useSessionStorage<boolean>("DOZZLE_SWARM_MODE", false);

watch(
  route,
  () => {
    if (route.meta.swarmMode) {
      showSwarm.value = true;
    } else if (route.meta.containerMode) {
      showSwarm.value = false;
    }
  },
  { immediate: true },
);
</script>
<style scoped lang="postcss"></style>
