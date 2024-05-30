<template>
  <Search />
  <ServiceLog :name="name" :scrollable="pinnedLogs.length > 0" />
</template>

<script lang="ts" setup>
const { name } = defineProps<{ name: string }>();

const containerStore = useContainerStore();
const { ready } = storeToRefs(containerStore);

const pinnedLogsStore = usePinnedLogsStore();
const { pinnedLogs } = storeToRefs(pinnedLogsStore);

const stackStore = useSwarmStore();
const { services } = storeToRefs(stackStore);
const service = computed(() => services.value.find((s) => s.name === name));

watchEffect(() => {
  if (ready.value) {
    if (service.value?.name) {
      setTitle(service.value.name);
    } else {
      setTitle("Not Found");
    }
  }
});
</script>
<route lang="yaml">
meta:
  swarmMode: true
</route>
