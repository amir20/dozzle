<template>
  <div v-if="ready" data-testid="side-menu" class="flex min-h-0 flex-col">
    <Carousel v-model="selectedCard" class="flex-1">
      <CarouselItem :title="$t('label.host-menu')" id="host">
        <HostMenu />
      </CarouselItem>
      <CarouselItem :title="$t('label.group-menu')" v-if="customGroups.length > 0" id="group">
        <GroupMenu />
      </CarouselItem>
      <CarouselItem :title="$t('label.swarm-menu')" v-if="services.length > 0" id="swarm">
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
const selectedCard = ref<"host" | "swarm" | "group">("host");

watch(
  route,
  () => {
    if (route.meta.menu && ["host", "swarm", "group"].includes(route.meta.menu as string)) {
      selectedCard.value = route.meta.menu as "host" | "swarm" | "group";
    }
  },
  { immediate: true },
);
</script>
<style scoped></style>
