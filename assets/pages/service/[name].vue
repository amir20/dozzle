<template>
  <Search />
  <ServiceLog :name="name" :scrollable="activeContainers.length > 0" />
</template>

<script lang="ts" setup>
const { name } = defineProps<{ name: string }>();

const containerStore = useContainerStore();
const { activeContainers, ready } = storeToRefs(containerStore);

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
