<template>
  <div class="group/item clickable relative flex w-full gap-x-2" @click="showDrawer(LogDetails, { entry: logEntry })">
    <div v-if="showContainerName">
      <ContainerName :id="logEntry.containerID" />
    </div>
    <div v-if="showStd">
      <LogStd :std="logEntry.std" />
    </div>
    <div v-if="showTimestamp">
      <LogDate :date="logEntry.date" class="select-none" />
    </div>
    <div class="flex">
      <LogLevel :level="logEntry.level" />
    </div>
    <div>
      <ul class="fields space-x-4">
        <li v-for="(value, name) in validValues" :key="name">
          <span class="text-light">{{ name }}=</span><span class="font-bold" v-if="value === null">&lt;null&gt;</span>
          <template v-else-if="Array.isArray(value)">
            <span class="font-bold" v-html="JSON.stringify(value)"> </span>
          </template>
          <span class="font-bold" v-html="value" v-else></span>
        </li>
        <li class="text-light" v-if="Object.keys(validValues).length === 0">all values are hidden</li>
      </ul>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { type ComplexLogEntry } from "@/models/LogEntry";
import LogDetails from "./LogDetails.vue";

const { logEntry, showContainerName = false } = defineProps<{
  logEntry: ComplexLogEntry;
  showContainerName?: boolean;
}>();

const validValues = computed(() => {
  return Object.fromEntries(Object.entries(logEntry.message).filter(([_, value]) => value !== undefined));
});

const showDrawer = useDrawer();
</script>

<style lang="postcss" scoped>
.text-light {
  @apply text-base-content/70;
}

.fields {
  li {
    @apply inline-flex;
  }
}
</style>
