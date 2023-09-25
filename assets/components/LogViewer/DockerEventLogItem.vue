<template>
  <div class="flex-1 font-sans text-[0.9rem]">
    <span class="whitespace-pre-wrap" :data-event="logEntry.event" v-html="logEntry.message"></span>
    <div
      class="alert alert-info mt-8 w-auto text-[1rem] md:mx-auto md:w-1/2"
      v-if="nextContainer && logEntry.event === 'container-stopped'"
    >
      <carbon:information class="h-6 w-6 shrink-0 stroke-current" />
      <span>
        Another container instance with the same name was created <distance-time :date="nextContainer.created" />. Do
        you want to redirect to the new one?
      </span>
      <div>
        <router-link :to="{ name: 'container-id', params: { id: nextContainer.id } }" class="btn btn-primary btn-sm">
          Redirect
        </router-link>
      </div>
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
      .filter(
        (c) =>
          c.host === container.value.host &&
          c.created > logEntry.date &&
          c.name === container.value.name &&
          c.state === "running",
      )
      .toSorted((a, b) => +a.created - +b.created)[0],
);
</script>

<style lang="postcss" scoped>
[data-event="container-stopped"] {
  @apply text-red;
}
[data-event="container-started"] {
  @apply text-green;
}
</style>
