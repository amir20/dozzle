<template>
  <div v-if="ready" data-testid="side-menu" class="flex min-h-0 w-full flex-col">
    <Carousel v-model="selectedCard" class="flex-1">
      <CarouselItem v-if="config.mode === 'k8s'" :title="$t('label.k8s-menu')" id="k8s">
        <K8sMenu />
      </CarouselItem>
      <CarouselItem v-if="config.mode === 'swarm' && services.length > 0" :title="$t('label.swarm-menu')" id="swarm">
        <SwarmMenu />
      </CarouselItem>
      <CarouselItem :title="$t('label.host-menu')" id="host">
        <HostMenu />
      </CarouselItem>
      <CarouselItem :title="$t('label.group-menu')" v-if="customGroups.length > 0" id="group">
        <GroupMenu />
      </CarouselItem>
      <CarouselItem v-if="config.mode !== 'swarm' && services.length > 0" :title="$t('label.swarm-menu')" id="swarm">
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

let defaultCard: "host" | "swarm" | "group" | "k8s";
switch (config.mode) {
  case "k8s":
    defaultCard = "k8s";
    break;
  case "swarm":
    defaultCard = "swarm";
    break;
  default:
    defaultCard = "host";
}
const selectedCard = ref<"host" | "swarm" | "group" | "k8s">(defaultCard);

watch(
  route,
  () => {
    if (route.meta.menu && ["host", "swarm", "group", "k8s"].includes(route.meta.menu as string)) {
      selectedCard.value = route.meta.menu as "host" | "swarm" | "group" | "k8s";
    }
  },
  { immediate: true },
);
</script>
<style scoped></style>
