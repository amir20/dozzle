<template>
  <Search />
  <StackLog :name="route.params.name" :scrollable="splitColumns" />
</template>

<script lang="ts" setup>
const route = useRoute("/stack/[name]");

const containerStore = useContainerStore();
const { ready } = storeToRefs(containerStore);

const splitColumns = useSplitColumns();

const stackStore = useSwarmStore();
const { stacks } = storeToRefs(stackStore);
const stack = computed(() => stacks.value.find((s) => s.name === route.params.name));

watchEffect(() => {
  if (ready.value) {
    if (stack.value?.name) {
      setTitle(stack.value.name);
    } else {
      setTitle("Not Found");
    }
  }
});
</script>
<route lang="yaml">
meta:
  menu: swarm
</route>
