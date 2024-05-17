<template>
  <Search />
  <StackLog :name="name" :scrollable="activeContainers.length > 0" />
</template>

<script lang="ts" setup>
import { Stack } from "@/models/Stack";

const { name } = defineProps<{ name: string }>();

const containerStore = useContainerStore();
const { activeContainers, ready } = storeToRefs(containerStore);

const stackStore = useStackStore();
const { stacks } = storeToRefs(stackStore);
const stack = computed(() => stacks.value.find((s) => s.name === name) ?? new Stack("", []));

watchEffect(() => {
  if (ready.value) {
    if (stack.value.name) {
      setTitle(stack.value.name);
    } else {
      setTitle("Not Found");
    }
  }
});
</script>
