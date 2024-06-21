<template>
  <div v-if="ready" data-testid="side-menu">
    <Toggle v-model="showSwarm" v-if="services.length > 0 || customGroups.length > 0">
      <div class="text-lg font-light">{{ $t("label.swarm-mode") }}</div>
    </Toggle>

    <SlideTransition :slide-right="showSwarm">
      <template #left>
        <HostMenu />
      </template>
      <template #right>
        <SwarmMenu />
      </template>
    </SlideTransition>
  </div>
  <div role="status" class="flex animate-pulse flex-col gap-4" v-else>
    <div class="h-3 w-full rounded-full bg-base-content/50 opacity-50" v-for="_ in 9"></div>
    <span class="sr-only">Loading...</span>
  </div>
</template>

<script lang="ts" setup>
import { onBeforeRouteLeave } from "vue-router";
const containerStore = useContainerStore();
const { ready } = storeToRefs(containerStore);

const swarmStore = useSwarmStore();
const { services, customGroups } = storeToRefs(swarmStore);

const showSwarm = useSessionStorage<boolean>("DOZZLE_SWARM_MODE", false);

onBeforeRouteLeave((to) => {
  if (to.meta.swarmMode) {
    showSwarm.value = true;
  } else if (to.meta.containerMode) {
    showSwarm.value = false;
  }
});
</script>
<style scoped lang="postcss"></style>
