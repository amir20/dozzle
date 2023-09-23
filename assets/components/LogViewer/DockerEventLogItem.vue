<template>
  <div>
    <span class="whitespace-pre-wrap" :data-event="logEntry.event" v-html="logEntry.message"></span>
    <div v-if="nextContainer" class="bg-base-lighter p-2">
      Similar container found
      <router-link
        :to="{ name: 'container-id', params: { id: nextContainer.id } }"
        class="btn btn-primary btn-sm font-sans"
      >
        {{ nextContainer.name }}
      </router-link>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { DockerEventLogEntry } from "@/models/LogEntry";

const { logEntry } = defineProps<{
  logEntry: DockerEventLogEntry;
}>();

const store = useContainerStore();
const { containers } = storeToRefs(store);
const { container } = useContainerContext();

const nextContainer = computed(
  () =>
    containers.value
      .filter((c) => c.host === container.value.host)
      .filter((c) => c.created > logEntry.date)
      .filter((c) => c.storageKey === container.value.storageKey)
      .sort((a, b) => +a.created - +b.created)[0],
);

watchEffect(() => {
  if (nextContainer.value) {
    console.log(nextContainer.value);
  }
});
</script>

<style lang="postcss" scoped>
[data-event="container-stopped"] {
  @apply text-red;
}
[data-event="container-started"] {
  @apply text-green;
}
</style>
