<template>
  <Search />
  <StackLog :name="name" :scrollable="pinnedLogs.length > 0" />
</template>

<script lang="ts" setup>
const { name } = defineProps<{ name: string }>();

const containerStore = useContainerStore();
const { ready } = storeToRefs(containerStore);

const pinnedLogsStore = usePinnedLogsStore();
const { pinnedLogs } = storeToRefs(pinnedLogsStore);

const stackStore = useSwarmStore();
const { stacks } = storeToRefs(stackStore);
const stack = computed(() => stacks.value.find((s) => s.name === name));

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
